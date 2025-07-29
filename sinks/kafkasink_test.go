package sinks

import (
	"encoding/json"
	"testing"

	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestKafkaSink_UpdateEvents_SyncProducer(t *testing.T) {
	// Set up the mock sync producer
	mockProducer := mocks.NewSyncProducer(t, nil)
	defer func() {
		require.NoError(t, mockProducer.Close(), "mockProducer.Close() error")
	}()

	// Create a KafkaSink with the mock producer
	kafkaSink := &KafkaSink{
		Topic:    "test-topic",
		producer: mockProducer,
	}

	// Define events
	eNew := &v1.Event{
		InvolvedObject: v1.ObjectReference{
			Name: "test-object",
		},
		Message: "This is a new event",
	}
	eOld := &v1.Event{}

	// Define the expected message
	expectedMessage := NewEventData(eNew, eOld)
	expectedMessageJSON, _ := json.Marshal(expectedMessage)

	// Set expectations on the mock producer
	mockProducer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
		require.JSONEq(t, string(expectedMessageJSON), string(val))
		return nil // Return nil as there is no error to return
	})

	// Call the method under test
	kafkaSink.UpdateEvents(eNew, eOld)
}

func TestKafkaSink_UpdateEvents_AsyncProducer(t *testing.T) {
	// Set up the mock async producer
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer func() {
		require.NoError(t, mockProducer.Close(), "mockProducer.Close() error")
	}()

	// Create a KafkaSink with the mock producer
	kafkaSink := &KafkaSink{
		Topic:    "test-topic",
		producer: mockProducer,
	}

	// Define events
	eNew := &v1.Event{
		InvolvedObject: v1.ObjectReference{
			Name: "test-object",
		},
		Message: "This is a new event",
	}
	eOld := &v1.Event{}

	// Define the expected message
	expectedMessage := NewEventData(eNew, eOld)
	expectedMessageJSON, _ := json.Marshal(expectedMessage)

	// Set expectations on the mock producer
	mockProducer.ExpectInputWithCheckerFunctionAndSucceed(func(val []byte) error {
		require.JSONEq(t, string(expectedMessageJSON), string(val))
		return nil // Return nil as there is no error to return
	})

	// Call the method under test
	kafkaSink.UpdateEvents(eNew, eOld)
}
