package sinks

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestUpdateEvents_stdoutsink(t *testing.T) {
	sink := NewStdoutSink()
	sink.UpdateEvents(&corev1.Event{}, &corev1.Event{})
}
