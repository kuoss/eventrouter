package sinks

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
)

// MockSyncProducer is a mock implementation of sarama.SyncProducer
type MockSyncProducer struct {
	mock.Mock
}

func (m *MockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	args := m.Called(msg)
	return args.Get(0).(int32), args.Get(1).(int64), args.Error(2)
}

func (m *MockSyncProducer) SendMessages([]*sarama.ProducerMessage) error {
	return nil
}

func (m *MockSyncProducer) Close() error {
	return nil
}

// MockAsyncProducer is a mock implementation of sarama.AsyncProducer
type MockAsyncProducer struct {
	mock.Mock
	input  chan *sarama.ProducerMessage
	errors chan *sarama.ProducerError
}

func NewMockAsyncProducer() *MockAsyncProducer {
	return &MockAsyncProducer{
		input:  make(chan *sarama.ProducerMessage),
		errors: make(chan *sarama.ProducerError),
	}
}

func (m *MockAsyncProducer) AsyncClose() {}

func (m *MockAsyncProducer) Close() error { return nil }

func (m *MockAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	return m.input
}

func (m *MockAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return nil
}

func (m *MockAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return m.errors
}

func TestKafkaSink_UpdateEvents_SyncProducer(t *testing.T) {
	mockProducer := new(MockSyncProducer)
	kafkaSink := &KafkaSink{
		Topic:    "test-topic",
		producer: mockProducer,
	}

	event := &v1.Event{InvolvedObject: v1.ObjectReference{Name: "test-object"}}

	mockProducer.On("SendMessage", mock.Anything).Return(int32(0), int64(0), nil)

	kafkaSink.UpdateEvents(event, nil)

	mockProducer.AssertCalled(t, "SendMessage", mock.Anything)
}

func TestKafkaSink_UpdateEvents_AsyncProducer(t *testing.T) {
	mockProducer := NewMockAsyncProducer()
	kafkaSink := &KafkaSink{
		Topic:    "test-topic",
		producer: mockProducer,
	}

	event := &v1.Event{InvolvedObject: v1.ObjectReference{Name: "test-object"}}

	// Verify message is sent to the input channel
	go func() {
		msg := <-mockProducer.input
		assert.NotNil(t, msg)
		assert.Equal(t, "test-object", string(msg.Key.(sarama.StringEncoder)))
	}()

	kafkaSink.UpdateEvents(event, nil)
}
