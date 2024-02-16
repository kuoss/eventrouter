package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestDeleteEvent(t *testing.T) {

	er := EventRouter{}

	testCases := []struct {
		obj interface{}
	}{
		// *v1.Event
		{&v1.Event{}},
		{&v1.Event{Reason: "hello", Message: "world"}},

		// *cache.DeletedFinalStateUnknown
		{&cache.DeletedFinalStateUnknown{}},
		{&cache.DeletedFinalStateUnknown{Key: "hello", Obj: "world"}},

		// others
		{v1.Event{}},
		{v1.Event{Reason: "hello", Message: "world"}},
		{cache.DeletedFinalStateUnknown{}},
		{cache.DeletedFinalStateUnknown{Key: "hello", Obj: "world"}},
		{v1.Pod{}},
		{&v1.Pod{}},
		{nil},
		{"string"},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("#%d_%T", i, tc.obj), func(t *testing.T) {
			require.NotPanics(t, func() {
				er.deleteEvent(tc.obj)
			})
		})
	}
}

func TestToEventPointer(t *testing.T) {
	testCases := []struct {
		obj       interface{}
		wantEvent *v1.Event
		wantError string
	}{
		// *v1.Event
		{
			&v1.Event{},
			&v1.Event{}, "",
		},
		{
			&v1.Event{Reason: "hello", Message: "world"},
			&v1.Event{Reason: "hello", Message: "world"}, "",
		},
		// *cache.DeletedFinalStateUnknown
		{
			&cache.DeletedFinalStateUnknown{},
			nil, "unexpected type: *cache.DeletedFinalStateUnknown",
		},
		{
			&cache.DeletedFinalStateUnknown{Key: "hello", Obj: "world"},
			nil, "unexpected type: *cache.DeletedFinalStateUnknown",
		},
		// others
		{
			v1.Event{},
			nil, "unexpected type: v1.Event",
		},
		{
			v1.Event{Reason: "hello", Message: "world"},
			nil, "unexpected type: v1.Event",
		},
		{
			cache.DeletedFinalStateUnknown{},
			nil, "unexpected type: cache.DeletedFinalStateUnknown",
		},
		{
			cache.DeletedFinalStateUnknown{Key: "hello", Obj: "world"},
			nil, "unexpected type: cache.DeletedFinalStateUnknown",
		},
		{
			v1.Pod{},
			nil, "unexpected type: v1.Pod",
		},
		{
			&v1.Pod{},
			nil, "unexpected type: *v1.Pod",
		},
		{
			nil,
			nil, "unexpected type: <nil>",
		},
		{
			"string",
			nil, "unexpected type: string",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("#%d_%v", i, tc.obj), func(t *testing.T) {
			e, err := toEventPointer(tc.obj)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantError)
			}
			require.Equal(t, tc.wantEvent, e)
		})
	}
}
