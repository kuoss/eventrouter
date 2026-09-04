package kubeevent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	created  = time.Date(2026, 9, 4, 6, 30, 0, 0, time.UTC)
	first    = time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)
	last     = time.Date(2026, 9, 4, 6, 40, 0, 0, time.UTC)
	observed = time.Date(2026, 9, 4, 6, 45, 0, 0, time.UTC)
)

// coreAPIEvent is an event as a core/v1 reporter writes it: kubelet still uses
// the legacy recorder, so source and the timestamps are filled in and the
// reporting fields are empty.
func coreAPIEvent() *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod.18d1b5694b849758",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: created},
		},
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "test-pod", Namespace: "default"},
		Reason:         "Started",
		Message:        "Started container app",
		Source:         corev1.EventSource{Component: "kubelet", Host: "node-1"},
		FirstTimestamp: metav1.Time{Time: first},
		LastTimestamp:  metav1.Time{Time: last},
		Count:          2,
		Type:           "Normal",
	}
}

// eventsAPIEvent is the same shape the API server returns over core/v1 for an
// event that its reporter wrote through events.k8s.io/v1 - the scheduler, for
// one. source, firstTimestamp, lastTimestamp and count all come back empty;
// eventTime and the reporting fields carry the information instead. The values
// are copied from a real event on a v1.36 cluster.
func eventsAPIEvent() *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "windows-exporter-7w6wh.18d20aa86bd78a46",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Time{Time: created},
		},
		InvolvedObject:      corev1.ObjectReference{Kind: "Pod", Name: "windows-exporter-7w6wh", Namespace: "kube-system"},
		Reason:              "Scheduled",
		Message:             "Successfully assigned kube-system/windows-exporter-7w6wh to node-1",
		EventTime:           metav1.MicroTime{Time: first},
		Action:              "Binding",
		ReportingController: "default-scheduler",
		ReportingInstance:   "default-scheduler-kube-scheduler-7b4d95d8bc-9gv7t",
		Type:                "Normal",
	}
}

func TestComponent(t *testing.T) {
	testCases := []struct {
		name  string
		event *corev1.Event
		want  string
	}{
		{
			name:  "core/v1 event reports its source component",
			event: coreAPIEvent(),
			want:  "kubelet",
		},
		{
			name:  "events.k8s.io/v1 event falls back to the reporting controller",
			event: eventsAPIEvent(),
			want:  "default-scheduler",
		},
		{
			name: "source wins when both are set",
			event: &corev1.Event{
				Source:              corev1.EventSource{Component: "kubelet"},
				ReportingController: "default-scheduler",
			},
			want: "kubelet",
		},
		{
			name:  "empty when the event names neither",
			event: &corev1.Event{},
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Component(tc.event))
		})
	}
}

func TestHost(t *testing.T) {
	testCases := []struct {
		name  string
		event *corev1.Event
		want  string
	}{
		{
			name:  "core/v1 event reports its source host",
			event: coreAPIEvent(),
			want:  "node-1",
		},
		{
			// reportingInstance is an instance ID, not a hostname, so it is
			// deliberately not used as one.
			name:  "events.k8s.io/v1 event has no host to report",
			event: eventsAPIEvent(),
			want:  "",
		},
		{
			name:  "empty when the event names none",
			event: &corev1.Event{},
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Host(tc.event))
		})
	}
}

func TestTimestamp(t *testing.T) {
	testCases := []struct {
		name  string
		event *corev1.Event
		want  time.Time
	}{
		{
			name:  "core/v1 event uses its last timestamp",
			event: coreAPIEvent(),
			want:  last,
		},
		{
			name:  "events.k8s.io/v1 singleton falls back to the event time",
			event: eventsAPIEvent(),
			want:  first,
		},
		{
			name: "a series prefers the last observed time over the event time",
			event: &corev1.Event{
				EventTime: metav1.MicroTime{Time: first},
				Series: &corev1.EventSeries{
					Count:            3,
					LastObservedTime: metav1.MicroTime{Time: observed},
				},
			},
			want: observed,
		},
		{
			name: "an empty series does not shadow the event time",
			event: &corev1.Event{
				EventTime: metav1.MicroTime{Time: first},
				Series:    &corev1.EventSeries{},
			},
			want: first,
		},
		{
			name:  "the first timestamp is used when nothing newer is set",
			event: &corev1.Event{FirstTimestamp: metav1.Time{Time: first}},
			want:  first,
		},
		{
			name: "the creation timestamp is the last resort",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: created}},
			},
			want: created,
		},
		{
			name:  "zero for an event that carries no time at all",
			event: &corev1.Event{},
			want:  time.Time{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Timestamp(tc.event))
		})
	}
}
