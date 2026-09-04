package sinks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// Mock server for InfluxDB
func setupMockServer() (*httptest.Server, InfluxDBSinkInterface, func()) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v2/write", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mockServer := httptest.NewServer(handler)

	config := InfluxdbConfig{
		Host: mockServer.URL,
	}
	sink, _ := NewInfluxdbSink(config)

	return mockServer, sink, func() {
		mockServer.Close()
	}
}

// Test event conversion to point with fields
func TestEventToPointWithFields(t *testing.T) {
	_, _, teardown := setupMockServer()
	defer teardown()

	event := createTestEvent("success-test-event", "Succeeded", nil, nil)
	point, err := eventToPointWithFields(event)

	require.NoError(t, err)
	require.NotNil(t, point)
	require.Equal(t, "events", point.Name())
}

// Test event data successfully sent to InfluxDB
func TestSendDataToInfluxDB(t *testing.T) {
	_, sink, teardown := setupMockServer()
	defer teardown()

	// Send valid data
	event := createTestEvent("test-event", "Succeeded", nil, nil)
	point, err := eventToPoint(event)
	require.NoError(t, err)

	// Using a goroutine-safe approach for client operations
	go func() {
		sink.sendData([]*write.Point{point})
	}()
}

// Simulate a server connection error
func TestServerConnectionError(t *testing.T) {
	badConfig := InfluxdbConfig{Host: "http://nonexistent:8086"}
	sink, _ := NewInfluxdbSink(badConfig)

	event := createTestEvent("failed-event", "Failed", nil, nil)
	point, err := eventToPointWithFields(event)
	require.NoError(t, err)

	go func() {
		sink.sendData([]*write.Point{point})
	}()
}

func pointTags(point *write.Point) map[string]string {
	tags := map[string]string{}
	for _, tag := range point.TagList() {
		tags[tag.Key] = tag.Value
	}
	return tags
}

// TestEventToPointTagsAndTime pins down the two fields that differ between the
// event APIs: the component tag and the point timestamp. Reading lastTimestamp
// off an events.k8s.io/v1 event would write the point at the start of the
// epoch, and reading source.component would leave it unattributed.
func TestEventToPointTagsAndTime(t *testing.T) {
	tm := time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)

	testCases := []struct {
		name          string
		event         *corev1.Event
		wantComponent string
		wantHostname  string
	}{
		{
			name:          "core/v1 event",
			event:         createTestEvent("test-event", "Started", &tm, &tm),
			wantComponent: "kubelet",
			wantHostname:  "node-1",
		},
		{
			name:          "events.k8s.io/v1 event",
			event:         createTestEventsAPIEvent("test-event", "Scheduled", tm),
			wantComponent: "default-scheduler",
			wantHostname:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withFields, err := eventToPointWithFields(tc.event)
			require.NoError(t, err)
			require.Equal(t, tm, withFields.Time())
			tags := pointTags(withFields)
			require.Equal(t, tc.wantComponent, tags["component"])
			require.Equal(t, tc.wantHostname, tags[LabelHostname.Key])

			point, err := eventToPoint(tc.event)
			require.NoError(t, err)
			require.Equal(t, tm, point.Time())
			require.Equal(t, tc.wantHostname, pointTags(point)[LabelHostname.Key])
		})
	}
}
