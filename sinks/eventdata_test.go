package sinks

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/rfc5424"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestEvent() *v1.Event {
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: v1.ObjectReference{
			Kind: "Pod",
			Name: "test-pod",
		},
		Reason:         "Scheduled",
		Message:        "Successfully assigned test-pod to node-1",
		Source:         v1.EventSource{Component: "scheduler", Host: "node-1"},
		FirstTimestamp: metav1.Time{Time: time.Now()},
		LastTimestamp:  metav1.Time{Time: time.Now()},
		Type:           "Normal",
	}
}

func TestWriteRFC5424(t *testing.T) {
	event := createTestEvent()
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteRFC5424(&buffer)

	assert.NoError(t, err)
	assert.NotEmpty(t, buffer.String())

	// Check if the output is a valid RFC5424 message
	rfcMessage := buffer.String()
	parts := strings.SplitN(rfcMessage, " ", 2)
	assert.NotEmpty(t, parts[0])
	assert.True(t, len(parts) > 1)

	message := rfc5424.Message{}
	err = message.UnmarshalBinary([]byte(parts[1]))

	assert.NoError(t, err)
	assert.Equal(t, event.Source.Host, message.Hostname)
	assert.Equal(t, event.Source.Component, message.AppName)
}

func TestWriteFlattenedJSON(t *testing.T) {
	event := createTestEvent()
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteFlattenedJSON(&buffer)

	assert.NoError(t, err)
	assert.NotEmpty(t, buffer.String())

	// Check if JSON is flattened into snake case format
	output := buffer.String()
	assert.Contains(t, output, `"event_involvedObject_kind":`)
	assert.Contains(t, output, `"event_metadata_namespace":"default"`)
	assert.Contains(t, output, `"verb":"ADDED"`)
}
