/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sinks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ref "k8s.io/client-go/tools/reference"

	corev1 "k8s.io/api/core/v1"
)

func TestUpdateEvents_httpsink(t *testing.T) {
	stopCh := make(chan bool, 1)
	doneCh := make(chan bool, 1)

	// The handler below runs on the server's own goroutine, so everything it
	// shares with the test body is guarded by mu.
	var mu sync.Mutex
	got := bytes.NewBuffer(nil)
	seenRequests := make([]*http.Request, 0)
	mockStatus := http.StatusOK

	setStatus := func(status int) {
		mu.Lock()
		defer mu.Unlock()
		mockStatus = status
	}
	body := func() string {
		mu.Lock()
		defer mu.Unlock()
		return got.String()
	}
	requestCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(seenRequests)
	}
	// requestReceived fires once per request the handler below finishes
	// recording, so the test can wait for an event to actually reach the
	// server instead of guessing how long that takes with a sleep. Buffered
	// well past any single scenario's request count so the handler never
	// blocks on it.
	requestReceived := make(chan struct{}, 100)

	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		got.Truncate(0)
		seenRequests = seenRequests[:0]
		// Discard any signal a previous scenario's request(s) left unclaimed
		// (a passing scenario doesn't necessarily call waitForRequest once
		// per request it triggers), so the next waitForRequest can't return
		// early on a stale one.
		for {
			select {
			case <-requestReceived:
			default:
				return
			}
		}
	}

	waitForRequest := func() {
		t.Helper()
		select {
		case <-requestReceived:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for a request to reach the test server")
		}
	}

	// Make a test server to send stuff too... it just copies its input to the
	// `got` buffer and records the request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		seenRequests = append(seenRequests, r)
		written, err := io.Copy(got, r.Body)
		// assert, not require: FailNow must not be called off the test goroutine.
		assert.NoError(t, err)
		assert.NotEmpty(t, written)
		w.WriteHeader(mockStatus)
		requestReceived <- struct{}{}
	}))
	defer srv.Close()

	testPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "baz",
			UID:       "bar",
		},
		Spec: corev1.PodSpec{},
	}
	podRef, err := ref.GetReference(scheme.Scheme, testPod)
	require.NoError(t, err)

	evt := makeFakeEvent(podRef, corev1.EventTypeWarning, "CreateInCluster", "Fake pod creation event")

	// 1. Try with a synchronous channel
	sink := NewHTTPSink(srv.URL, false, 0)
	go func() {
		sink.Run(stopCh)
		doneCh <- true
	}()

	// Send the event
	sink.UpdateEvents(evt, nil)
	stopCh <- true
	<-doneCh

	if body() == "" {
		t.Errorf("Sent logs but didn't read any back")
	}

	// 2. Try with the server returning 500's, test retries
	reset()
	sink = NewHTTPSink(srv.URL, false, 10)

	go func() {
		sink.Run(stopCh)
		doneCh <- true
	}()

	// Send the event, and wait for the first attempt to actually land before
	// flipping the server back to healthy - a buffered channel accepts
	// UpdateEvents before Run has necessarily claimed it, so there is no
	// other reliable signal that a request is even in flight yet.
	setStatus(http.StatusInternalServerError)
	sink.UpdateEvents(evt, nil)
	waitForRequest()
	setStatus(http.StatusOK)

	// Start the server, then send the stop chan. Since it's synchronous, the HTTP
	// client should still be trying to retry, so it won't read from the stop chan
	// again until it's finished retrying
	stopCh <- true
	<-doneCh

	if body() == "" {
		t.Errorf("Sent logs but didn't read any back. HTTP error log: %v", sink.httpClient.Error)
	}
	if requestCount() < 2 {
		t.Errorf("Tried to simulate server errors for retry, more than one request should have been sent")
	}

	// 3. Try with an overflowing channel, write a bunch of events out, only 10
	// should be consumed (the rest discarded, since we're not running the
	// processing loop yet.)
	numExpected := 10
	reset()
	sink = NewHTTPSink(srv.URL, true, numExpected)

	for i := 0; i < 1000; i++ {
		evt.Message = "msg " + strconv.Itoa(i)
		sink.UpdateEvents(evt, nil)
	}

	go func() {
		sink.Run(stopCh)
		doneCh <- true
	}()

	// Wait for the coalesced batch to actually be posted rather than
	// guessing how long that takes.
	waitForRequest()

	stopCh <- true
	<-doneCh

	newlines := strings.Count(body(), "\n")
	if newlines != numExpected {
		t.Errorf("Got wrong number of lines back (got %v, expected %v)", newlines, numExpected)
	}
	if requestCount() > 1 {
		t.Errorf("Pending logs should have been coalesced into one request, but got %v requests", requestCount())
	}
}

func makeFakeEvent(ref *corev1.ObjectReference, eventtype, reason, message string) *corev1.Event {
	tm := metav1.Time{
		Time: time.Now(),
	}
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", ref.Name, tm.UnixNano()),
			Namespace: ref.Namespace,
		},
		InvolvedObject: *ref,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: tm,
		LastTimestamp:  tm,
		Count:          1,
		Type:           eventtype,
	}
}
