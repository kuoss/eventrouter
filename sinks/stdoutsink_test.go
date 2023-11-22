package sinks

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestNewStdoutSink(t *testing.T) {
	testCases := []struct {
		namespace string
		want      *StdoutSink
	}{
		{"", &StdoutSink{namespace: ""}},
		{"hello", &StdoutSink{namespace: "hello"}},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got := NewStdoutSink(tc.namespace)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestUpdateEvents_stdoutsink(t *testing.T) {
	testCases := []struct {
		namespace string
		eNew      *v1.Event
		eOld      *v1.Event
	}{
		{"", &v1.Event{}, &v1.Event{}},
		{"foo", &v1.Event{}, &v1.Event{}},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			sink := NewStdoutSink(tc.namespace)
			sink.UpdateEvents(tc.eNew, tc.eOld)
		})
	}
}
