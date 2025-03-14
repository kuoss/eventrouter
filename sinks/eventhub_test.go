package sinks

import (
	"context"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/eapache/channels"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func init() {
	var hub *eventhub.Hub
	var _ EventHubClient = hub
}

// MockedHub is a mock implementation of the event hub
type MockedHub struct {
	mock.Mock
}

func (m *MockedHub) SendBatch(ctx context.Context, iterator eventhub.BatchIterator, opts ...eventhub.BatchOption) error {
	args := m.Called(ctx, iterator)
	return args.Error(0)
}

type MockEventHubClient struct {
	mock.Mock
}

func (m *MockEventHubClient) SendBatch(ctx context.Context, iterator eventhub.BatchIterator, opts ...eventhub.BatchOption) error {
	args := m.Called(ctx, iterator)
	return args.Error(0)
}

func TestNewEventHubSink(t *testing.T) {
	testCases := []struct {
		name       string
		connString string
		overflow   bool
		bufferSize int
		wantError  string
	}{
		{
			name:       "Valid connection string",
			connString: "Endpoint=sb://your-endpoint.servicebus.windows.net/;SharedAccessKeyName=your-access-key-name;SharedAccessKey=your-access-key;EntityPath=your-event-hub-name",
			overflow:   false,
			bufferSize: 10,
			wantError:  "",
		},
		{
			name:       "Invalid connection string",
			connString: "",
			overflow:   false,
			bufferSize: 10,
			wantError:  "failed parsing connection string due to unmatched key value separated by '='",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewEventHubSink(tc.connString, tc.overflow, tc.bufferSize)
			if tc.wantError == "" {
				require.NoError(t, err)
				require.NotNil(t, got)
			} else {
				require.EqualError(t, err, tc.wantError)
				require.Nil(t, got)
			}
		})
	}
}

func TestRun(t *testing.T) {
	mockHub := new(MockEventHubClient)
	eventBufferSize := 5

	// Create a buffered channel using eapache channels library
	eventCh := channels.NewNativeChannel(channels.BufferCap(eventBufferSize))
	sink := &EventHubSink{hub: mockHub, eventCh: eventCh}

	stopCh := make(chan bool)

	// Set expectation for SendBatch method
	mockHub.On("SendBatch", mock.Anything, mock.Anything).Return(nil).Once()

	// Run the EventHubSink in a separate goroutine
	go sink.Run(stopCh)

	// Send a test event to the channel
	sink.eventCh.In() <- NewEventData(&v1.Event{Message: "TestEvent"}, nil)

	// Allow some time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	// Signal the Run method to stop
	close(stopCh)

	// Assert that SendBatch was called once as expected
	mockHub.AssertExpectations(t)
}

func TestSendBatch(t *testing.T) {
	mockHub := new(MockedHub)
	sink := &EventHubSink{hub: mockHub}

	events := []*eventhub.Event{
		eventhub.NewEvent([]byte(`{"event":"event1"}`)),
		eventhub.NewEvent([]byte(`{"event":"event2"}`)),
	}

	// Set up the expectation
	mockHub.On("SendBatch", mock.Anything, mock.Anything).Return(nil).Once()

	// Call method to test
	sink.sendBatch(events)

	// Verify expectations
	mockHub.AssertExpectations(t)
}

func TestDrainEvents(t *testing.T) {
	mockHub := new(MockedHub)
	sink := &EventHubSink{hub: mockHub}

	events := []EventData{
		{Verb: "create", Event: &v1.Event{Message: "Event1"}},
		{Verb: "update", Event: &v1.Event{Message: "Event2"}, OldEvent: &v1.Event{Message: "OldEvent2"}},
	}

	// Set up the expectation that SendBatch is called once with any context and any batch iterator
	mockHub.On("SendBatch", mock.Anything, mock.Anything).Return(nil).Once()

	// Call drainEvents which we want to test
	sink.drainEvents(events)

	// Assert the expectations were met
	mockHub.AssertExpectations(t)
}
