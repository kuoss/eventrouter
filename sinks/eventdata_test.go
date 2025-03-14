package sinks

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestEvent(name, reason string) *v1.Event {
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       "12345",
		},
		InvolvedObject: v1.ObjectReference{
			Kind: "Pod",
			UID:  "pod12345",
		},
		Reason:         reason,
		Message:        "Successfully assigned test-pod to node-1",
		Source:         v1.EventSource{Component: "kubelet", Host: "node-1"},
		FirstTimestamp: metav1.Time{Time: time.Now()},
		LastTimestamp:  metav1.Time{Time: time.Now()},
		Type:           "Normal",
	}
}

func zeroDatetime(input string) string {
	re1 := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z`)
	output := re1.ReplaceAllStringFunc(input, func(s string) string {
		return "0000-00-00T00:00:00Z"
	})

	re2 := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?\+\d{2}:\d{2}`)
	output = re2.ReplaceAllStringFunc(output, func(s string) string {
		return "0000-00-00T00:00:00.000000000+00:00"
	})

	return output
}

func assertEqualIgnoreDatetime(t *testing.T, expected, actual string) {
	expectedZeroed := zeroDatetime(expected)
	actualZeroed := zeroDatetime(actual)
	assert.Equal(t, expectedZeroed, actualZeroed, "expected: %s, got: %s", expectedZeroed, actualZeroed)
}

func TestWriteRFC5424(t *testing.T) {
	want := `514 <24>1 2025-03-14T18:47:52.180226515+09:00 node-1 kubelet - - - {"verb":"ADDED","event":{"metadata":{"name":"test-event","namespace":"default","uid":"12345","creationTimestamp":null},"involvedObject":{"kind":"Pod","uid":"pod12345"},"reason":"Scheduled","message":"Successfully assigned test-pod to node-1","source":{"component":"kubelet","host":"node-1"},"firstTimestamp":"2025-03-14T09:47:52Z","lastTimestamp":"2025-03-14T09:47:52Z","type":"Normal","eventTime":null,"reportingComponent":"","reportingInstance":""}}`
	event := createTestEvent("test-event", "Scheduled")
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteRFC5424(&buffer)

	got := buffer.String()

	assert.NoError(t, err)
	assert.NotEmpty(t, got)
	assertEqualIgnoreDatetime(t, want, got)
}

func TestWriteFlattenedJSON(t *testing.T) {
	want := `{"event_eventTime":null,"event_firstTimestamp":"2000-01-01T00:00:00Z","event_involvedObject_kind":"Pod","event_involvedObject_uid":"pod12345","event_lastTimestamp":"2000-01-01T00:00:00Z","event_message":"Successfully assigned test-pod to node-1","event_metadata_creationTimestamp":null,"event_metadata_name":"test-event","event_metadata_namespace":"default","event_metadata_uid":"12345","event_reason":"Scheduled","event_reportingComponent":"","event_reportingInstance":"","event_source_component":"kubelet","event_source_host":"node-1","event_type":"Normal","verb":"ADDED"}`

	event := createTestEvent("test-event", "Scheduled")
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteFlattenedJSON(&buffer)
	assert.NoError(t, err)

	got := buffer.String()
	assertEqualIgnoreDatetime(t, want, got)
	assert.Contains(t, got, `"event_involvedObject_kind":`)
	assert.Contains(t, got, `"event_metadata_namespace":"default"`)
	assert.Contains(t, got, `"verb":"ADDED"`)
}
