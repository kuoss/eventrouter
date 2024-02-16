package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestDeleteEvent(t *testing.T) {
	er := EventRouter{}

	testCases := []struct {
		obj interface{}
	}{
		// normal cases
		{v1.Event{}},
		{v1.Event{Reason: "hello", Message: "world"}},
		// abnormal cases
		{&v1.Event{}},
		{v1.Pod{}},
		{&v1.Pod{}},
		{nil},
		{"string"},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			// all cases should not occur panic
			require.NotPanics(t, func() {
				er.deleteEvent(tc.obj)
			})
		})
	}
}
