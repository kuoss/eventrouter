package sinks

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/rfc5424"
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

func TestWriteRFC5424(t *testing.T) {
	event := createTestEvent("test-event", "Scheduled")
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
	want := `{"event_eventTime":null,"event_firstTimestamp":"2000-01-01T00:00:00Z","event_involvedObject_kind":"Pod","event_involvedObject_uid":"pod12345","event_lastTimestamp":"2000-01-01T00:00:00Z","event_message":"Successfully assigned test-pod to node-1","event_metadata_creationTimestamp":null,"event_metadata_name":"test-event","event_metadata_namespace":"default","event_metadata_uid":"12345","event_reason":"Scheduled","event_reportingComponent":"","event_reportingInstance":"","event_source_component":"kubelet","event_source_host":"node-1","event_type":"Normal","verb":"ADDED"}`

	event := createTestEvent("test-event", "Scheduled")
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteFlattenedJSON(&buffer)
	assert.NoError(t, err)
	timestampRegex := `"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z"`
	got := regexp.MustCompile(timestampRegex).ReplaceAllString(buffer.String(), `"2000-01-01T00:00:00Z"`)
	assert.Equal(t, want, got)
	assert.Contains(t, got, `"event_involvedObject_kind":`)
	assert.Contains(t, got, `"event_metadata_namespace":"default"`)
	assert.Contains(t, got, `"verb":"ADDED"`)
}
