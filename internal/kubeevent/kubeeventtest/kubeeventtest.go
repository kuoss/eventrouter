// Package kubeeventtest builds *corev1.Event values shaped the way the
// Kubernetes API server returns events reported through the core/v1 and the
// events.k8s.io/v1 API - see the internal/kubeevent package doc for why the
// two shapes differ. Tests across the module used to hand-roll their own
// copy of these two shapes; that let the copies drift, so this package is
// the single place that knows what each shape looks like. Every field a test
// case needs to vary is set through an Option; anything left unset keeps a
// realistic, self-consistent default.
package kubeeventtest

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// defaultTime is used for any timestamp an Option does not set. Its exact
// value carries no meaning to callers - it just needs to be non-zero.
var defaultTime = time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)

// Option overrides one field of the event CoreAPIEvent or EventsAPIEvent
// returns.
type Option func(*corev1.Event)

// WithName sets the event's own name (not the involved object's).
func WithName(name string) Option {
	return func(e *corev1.Event) { e.Name = name }
}

// WithNamespace sets the event's namespace.
func WithNamespace(namespace string) Option {
	return func(e *corev1.Event) { e.Namespace = namespace }
}

// WithUID sets the event's own UID.
func WithUID(uid string) Option {
	return func(e *corev1.Event) { e.UID = types.UID(uid) }
}

// WithInvolvedObject overrides the object the event is about. The default is
// a Pod named "test-pod" in namespace "default".
func WithInvolvedObject(obj corev1.ObjectReference) Option {
	return func(e *corev1.Event) { e.InvolvedObject = obj }
}

// WithReason sets the event's reason.
func WithReason(reason string) Option {
	return func(e *corev1.Event) { e.Reason = reason }
}

// WithMessage sets the event's message.
func WithMessage(message string) Option {
	return func(e *corev1.Event) { e.Message = message }
}

// WithType sets the event's type, e.g. "Normal" or "Warning".
func WithType(eventType string) Option {
	return func(e *corev1.Event) { e.Type = eventType }
}

// WithCount sets how many times the event has occurred.
func WithCount(count int32) Option {
	return func(e *corev1.Event) { e.Count = count }
}

// WithCreationTimestamp sets the object's creation time.
func WithCreationTimestamp(t time.Time) Option {
	return func(e *corev1.Event) { e.CreationTimestamp = metav1.Time{Time: t} }
}

// WithTimes sets a core/v1 event's firstTimestamp and lastTimestamp.
func WithTimes(first, last time.Time) Option {
	return func(e *corev1.Event) {
		e.FirstTimestamp = metav1.Time{Time: first}
		e.LastTimestamp = metav1.Time{Time: last}
	}
}

// WithEventTime sets an events.k8s.io/v1 event's eventTime.
func WithEventTime(t time.Time) Option {
	return func(e *corev1.Event) { e.EventTime = metav1.MicroTime{Time: t} }
}

// WithSeries sets an events.k8s.io/v1 event's series, the count/timestamp a
// repeated event carries in place of core/v1's plain Count/lastTimestamp.
func WithSeries(series *corev1.EventSeries) Option {
	return func(e *corev1.Event) { e.Series = series }
}

// CoreAPIEvent returns an event shaped the way a core/v1 reporter - kubelet,
// for one - writes it: source and both timestamps are set, and none of the
// reporting fields are.
func CoreAPIEvent(opts ...Option) *corev1.Event {
	e := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "test-pod", Namespace: "default"},
		Reason:         "Started",
		Source:         corev1.EventSource{Component: "kubelet", Host: "node-1"},
		FirstTimestamp: metav1.Time{Time: defaultTime},
		LastTimestamp:  metav1.Time{Time: defaultTime},
		Type:           "Normal",
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EventsAPIEvent returns the shape the API server returns over core/v1 for an
// event that its reporter wrote through events.k8s.io/v1 - the scheduler, for
// one: source, firstTimestamp, lastTimestamp and count all come back empty;
// eventTime and the reporting fields carry the information instead.
func EventsAPIEvent(opts ...Option) *corev1.Event {
	e := &corev1.Event{
		ObjectMeta:          metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		InvolvedObject:      corev1.ObjectReference{Kind: "Pod", Name: "test-pod", Namespace: "default"},
		Reason:              "Scheduled",
		EventTime:           metav1.MicroTime{Time: defaultTime},
		Action:              "Binding",
		ReportingController: "default-scheduler",
		ReportingInstance:   "default-scheduler-kube-scheduler-7b4d95d8bc-9gv7t",
		Type:                "Normal",
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
