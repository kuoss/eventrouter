package sinks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEventHubSink(t *testing.T) {
	testCases := []struct {
		connString string
		bufferSize int
		want       *EventHubSink
		wantError  string
	}{
		{"", 0, nil, `failed parsing connection string due to unmatched key value separated by '='`},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got, err := NewEventHubSink(tc.connString, tc.bufferSize)
			if tc.wantError == "" {
				require.NoError(t, err)
				require.NotEmpty(t, got)
			} else {
				require.EqualError(t, err, tc.wantError)
				require.Nil(t, got)
			}
		})
	}
}
