package sinks

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/kuoss/eventrouter/internal/kubeevent/kubeeventtest"
	"github.com/kuoss/eventrouter/sinks/rfc5424"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func createTestEvent(name, reason string, firstTime, lastTime *time.Time) *v1.Event {
	if firstTime == nil {
		firstTime = new(time.Now())
	}
	if lastTime == nil {
		lastTime = new(time.Now())
	}

	return kubeeventtest.CoreAPIEvent(
		kubeeventtest.WithName(name),
		kubeeventtest.WithUID("12345"),
		kubeeventtest.WithInvolvedObject(v1.ObjectReference{Kind: "Pod", UID: "pod12345"}),
		kubeeventtest.WithReason(reason),
		kubeeventtest.WithMessage("Successfully assigned test-pod to node-1"),
		kubeeventtest.WithTimes(*firstTime, *lastTime),
	)
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
	require.Equal(t, expectedZeroed, actualZeroed)
}

func TestWriteRFC5424(t *testing.T) {
	want := `489 <24>1 2024-03-15T12:34:56.123456789+09:00 node-1 kubelet - - - {"verb":"ADDED","event":{"metadata":{"name":"test-event","namespace":"default","uid":"12345"},"involvedObject":{"kind":"Pod","uid":"pod12345"},"reason":"Scheduled","message":"Successfully assigned test-pod to node-1","source":{"component":"kubelet","host":"node-1"},"firstTimestamp":"2025-03-14T09:47:52Z","lastTimestamp":"2025-03-14T09:47:52Z","type":"Normal","eventTime":null,"reportingComponent":"","reportingInstance":""}}`

	lastTime, _ := time.Parse(time.RFC3339Nano, "2024-03-15T12:34:56.123456789+09:00")
	event := createTestEvent("test-event", "Scheduled", nil, &lastTime)
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteRFC5424(&buffer)

	got := buffer.String()

	require.NoError(t, err)
	assertEqualIgnoreDatetime(t, want, got)
}

func TestWriteFlattenedJSON(t *testing.T) {
	want := `{"event_eventTime":null,"event_firstTimestamp":"2000-01-01T00:00:00Z","event_involvedObject_kind":"Pod","event_involvedObject_uid":"pod12345","event_lastTimestamp":"2000-01-01T00:00:00Z","event_message":"Successfully assigned test-pod to node-1","event_metadata_name":"test-event","event_metadata_namespace":"default","event_metadata_uid":"12345","event_reason":"Scheduled","event_reportingComponent":"","event_reportingInstance":"","event_source_component":"kubelet","event_source_host":"node-1","event_type":"Normal","verb":"ADDED"}`

	event := createTestEvent("test-event", "Scheduled", nil, nil)
	eventData := NewEventData(event, nil)

	var buffer bytes.Buffer
	_, err := eventData.WriteFlattenedJSON(&buffer)
	require.NoError(t, err)

	got := buffer.String()
	assertEqualIgnoreDatetime(t, want, got)
	require.Contains(t, got, `"event_involvedObject_kind":`)
	require.Contains(t, got, `"event_metadata_namespace":"default"`)
	require.Contains(t, got, `"verb":"ADDED"`)
}

// createTestEventsAPIEvent builds an event the way the API server returns one
// over core/v1 when its reporter wrote it through events.k8s.io/v1: no source
// and no first/last timestamp, with eventTime and the reporting fields
// carrying the information instead.
func createTestEventsAPIEvent(name, reason string, eventTime time.Time) *v1.Event {
	return kubeeventtest.EventsAPIEvent(
		kubeeventtest.WithName(name),
		kubeeventtest.WithUID("12345"),
		kubeeventtest.WithInvolvedObject(v1.ObjectReference{Kind: "Pod", UID: "pod12345"}),
		kubeeventtest.WithReason(reason),
		kubeeventtest.WithMessage("Successfully assigned default/test-pod to node-1"),
		kubeeventtest.WithEventTime(eventTime),
	)
}

// TestWriteRFC5424Header checks the syslog header for both flavours of event.
// The header is parsed back rather than compared as a string so that the
// timestamp is actually asserted - reading lastTimestamp off an
// events.k8s.io/v1 event would silently yield the zero time.
func TestWriteRFC5424Header(t *testing.T) {
	tm := time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)

	testCases := []struct {
		name         string
		event        *v1.Event
		wantHostname string
		wantAppName  string
	}{
		{
			name:         "core/v1 event",
			event:        createTestEvent("test-event", "Started", &tm, &tm),
			wantHostname: "node-1",
			wantAppName:  "kubelet",
		},
		{
			name:         "events.k8s.io/v1 event",
			event:        createTestEventsAPIEvent("test-event", "Scheduled", tm),
			wantHostname: "",
			wantAppName:  "default-scheduler",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eventData := NewEventData(tc.event, nil)

			var buffer bytes.Buffer
			_, err := eventData.WriteRFC5424(&buffer)
			require.NoError(t, err)

			msg, err := rfc5424.NewFromBytes(buffer.Bytes())
			require.NoError(t, err)
			require.True(t, tm.Equal(msg.Timestamp), "want %s, got %s", tm, msg.Timestamp)
			require.Equal(t, tc.wantHostname, msg.Hostname)
			require.Equal(t, tc.wantAppName, msg.AppName)
			require.Contains(t, msg.Message, `"reason":"`+tc.event.Reason+`"`)
		})
	}
}
