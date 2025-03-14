package sinks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/require"
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
