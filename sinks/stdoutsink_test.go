package sinks

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestUpdateEvents_stdoutsink(t *testing.T) {
	sink := NewStdoutSink()
	sink.UpdateEvents(&v1.Event{}, &v1.Event{})
}
