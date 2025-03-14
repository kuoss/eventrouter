package rfc5424

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBytes(t *testing.T) {
	want := "80 <24>1 2024-03-15T12:34:56.123456789+09:00 test-host test-app - - - Hello, world!"

	timestamp, _ := time.Parse(time.RFC3339Nano, "2024-03-15T12:34:56.123456789+09:00")
	msg := Message{
		Timestamp: timestamp,
		Hostname:  "test-host",
		AppName:   "test-app",
		Message:   "Hello, world!",
	}

	got := msg.Bytes()
	assert.Equal(t, want, string(got))
}

func TestNewFromBytes(t *testing.T) {
	input := "80 <24>1 2024-03-15T12:34:56.123456789+09:00 test-host test-app - - - Hello, world!"

	timestamp, _ := time.Parse(time.RFC3339Nano, "2024-03-15T12:34:56.123456789+09:00")
	want := &Message{Timestamp: timestamp, Hostname: "test-host", AppName: "test-app", Message: "Hello, world!"}

	got, err := NewFromBytes([]byte(input))
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
