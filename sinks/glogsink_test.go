package sinks

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGlogSink_UpdateEvents(t *testing.T) {
	sink := NewGlogSink()

	oldEvent := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-event",
		},
		Message: "This is an old event",
	}

	newEvent := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-event",
		},
		Message: "This is a new event",
	}

	// Call UpdateEvents method
	sink.UpdateEvents(newEvent, oldEvent)

}
