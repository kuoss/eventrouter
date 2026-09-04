package router

import (
	"fmt"
	"testing"

	"github.com/kuoss/eventrouter/internal/config"
	"time"

	"github.com/kuoss/eventrouter/internal/kubeevent/kubeeventtest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

var viper = testConfig{}

type testConfig struct{}

func (testConfig) Set(key string, value any) {
	if key == "enable-prometheus" {
		config.SetPrometheusForTest(value.(bool))
	}
}

var testTime = time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)

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
		// cache.DeletedFinalStateUnknown - client-go only ever constructs
		// this as a value (see tools/cache/delta_fifo.go), so this is what a
		// real informer actually hands DeleteFunc on a missed delete.
		{
			cache.DeletedFinalStateUnknown{Key: "default/foo", Obj: &v1.Event{Reason: "recovered"}},
			&v1.Event{Reason: "recovered"}, "",
		},
		{
			cache.DeletedFinalStateUnknown{},
			nil, "unexpected type: <nil>",
		},
		{
			cache.DeletedFinalStateUnknown{Key: "hello", Obj: "world"},
			nil, "unexpected type: string",
		},
		// *cache.DeletedFinalStateUnknown - a pointer, which client-go never
		// actually constructs. Not unwrapped, so this still falls through.
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

// coreAPIEvent is an event as a core/v1 reporter writes it: source and the
// timestamps are filled in, the reporting fields are empty.
func coreAPIEvent(eventType, reason string) *v1.Event {
	return kubeeventtest.CoreAPIEvent(
		kubeeventtest.WithName("test-pod.18d1b5694b849758"),
		kubeeventtest.WithReason(reason),
		kubeeventtest.WithTimes(testTime, testTime),
		kubeeventtest.WithType(eventType),
	)
}

// eventsAPIEvent is what the API server returns over core/v1 for an event its
// reporter wrote through events.k8s.io/v1: no source and no timestamps, with
// eventTime and the reporting fields carrying the information instead.
func eventsAPIEvent(eventType, reason string) *v1.Event {
	return kubeeventtest.EventsAPIEvent(
		kubeeventtest.WithName("test-pod.18d20aa86bd78a46"),
		kubeeventtest.WithReason(reason),
		kubeeventtest.WithEventTime(testTime),
		kubeeventtest.WithType(eventType),
	)
}

func TestPrometheusEvent(t *testing.T) {
	viper.Set("enable-prometheus", true)
	defer viper.Set("enable-prometheus", false)

	testCases := []struct {
		name          string
		event         *v1.Event
		wantVec       *prometheus.CounterVec
		wantSource    string
		wantComponent string
	}{
		{
			name:          "core/v1 event is labelled with its source",
			event:         coreAPIEvent("Normal", "Started"),
			wantVec:       kubernetesNormalEventCounterVec,
			wantSource:    "node-1",
			wantComponent: "kubelet",
		},
		{
			// Without the fallback both labels would be empty here.
			name:          "events.k8s.io/v1 event is labelled with its reporting controller",
			event:         eventsAPIEvent("Normal", "Scheduled"),
			wantVec:       kubernetesNormalEventCounterVec,
			wantSource:    "",
			wantComponent: "default-scheduler",
		},
		{
			name:          "warning goes to the warning counter",
			event:         coreAPIEvent("Warning", "BackOff"),
			wantVec:       kubernetesWarningEventCounterVec,
			wantSource:    "node-1",
			wantComponent: "kubelet",
		},
		{
			name:          "info goes to the info counter",
			event:         eventsAPIEvent("Info", "Informing"),
			wantVec:       kubernetesInfoEventCounterVec,
			wantSource:    "",
			wantComponent: "default-scheduler",
		},
		{
			name:          "an unknown type goes to the unknown counter",
			event:         coreAPIEvent("Surprising", "Surprise"),
			wantVec:       kubernetesUnknownEventCounterVec,
			wantSource:    "node-1",
			wantComponent: "kubelet",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			counter := tc.wantVec.WithLabelValues(
				tc.event.InvolvedObject.Kind,
				tc.event.InvolvedObject.Name,
				tc.event.InvolvedObject.Namespace,
				tc.event.Reason,
				tc.wantSource,
				tc.wantComponent,
			)
			before := testutil.ToFloat64(counter)

			prometheusEvent(tc.event)

			require.Equal(t, before+1, testutil.ToFloat64(counter))
		})
	}
}

func TestPrometheusEventDisabled(t *testing.T) {
	viper.Set("enable-prometheus", false)

	event := coreAPIEvent("Normal", "NotCounted")
	counter := kubernetesNormalEventCounterVec.WithLabelValues(
		"Pod", "test-pod", "default", "NotCounted", "node-1", "kubelet")

	prometheusEvent(event)

	require.Equal(t, 0.0, testutil.ToFloat64(counter))
}
