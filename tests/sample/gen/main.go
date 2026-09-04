// Command gen writes tests/sample/pod-log.ndjson: the JSON lines eventrouter's
// own logger emits during startup, interleaved with one core/v1 and one
// events.k8s.io/v1 event, each shown as both ADDED and UPDATED - in the shape
// `kubectl logs` shows once it merges the container's stdout and stderr. Run
// it with `make sample`; regenerate the file whenever a log message wording,
// or the EventData JSON shape, changes.
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kuoss/eventrouter/internal/kubeevent/kubeeventtest"
	"github.com/kuoss/eventrouter/sinks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sampleTime and sampleTimeLater stand in for wall-clock time so the
// generated file is the same on every run instead of re-diffing on its
// timestamps alone.
var (
	sampleTime      = time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)
	sampleTimeLater = sampleTime.Add(90 * time.Second)
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				a.Value = slog.StringValue(sampleTime.Format(time.RFC3339Nano))
			}
			return a
		},
	})))

	// eventrouter's own startup logs - see main.go, internal/router/router.go
	// and sinks/interfaces.go for where each of these is actually logged.
	slog.Info("starting eventrouter", "version", "dev")
	slog.Info("sink selected", "sink", "stdout")
	slog.Info("starting prometheus metrics", "address", ":8080")
	slog.Info("starting EventRouter")
	slog.Info("starting shared informer(s)")

	// A core/v1 event, first observed and then repeated: kubelet bumps Count
	// and lastTimestamp on the same object rather than sending a new one.
	coreAdded := kubeeventtest.CoreAPIEvent(
		kubeeventtest.WithName("my-app-7d9f8c6b4d-x2kqp.18f2a3b9c0d1e2f3"),
		kubeeventtest.WithNamespace("default"),
		kubeeventtest.WithUID("3f1e2a4b-1234-4a5b-9c6d-abcdef123456"),
		kubeeventtest.WithCreationTimestamp(sampleTime),
		kubeeventtest.WithInvolvedObject(corev1.ObjectReference{
			Kind: "Pod", Name: "my-app-7d9f8c6b4d-x2kqp", Namespace: "default",
			UID: "9c8b7a6f-5e4d-3c2b-1a09-fedcba987654",
		}),
		kubeeventtest.WithReason("BackOff"),
		kubeeventtest.WithMessage("Back-off restarting failed container app in pod my-app-7d9f8c6b4d-x2kqp"),
		kubeeventtest.WithType("Warning"),
		kubeeventtest.WithTimes(sampleTime, sampleTime),
		kubeeventtest.WithCount(1),
	)
	coreUpdated := kubeeventtest.CoreAPIEvent(
		kubeeventtest.WithName("my-app-7d9f8c6b4d-x2kqp.18f2a3b9c0d1e2f3"),
		kubeeventtest.WithNamespace("default"),
		kubeeventtest.WithUID("3f1e2a4b-1234-4a5b-9c6d-abcdef123456"),
		kubeeventtest.WithCreationTimestamp(sampleTime),
		kubeeventtest.WithInvolvedObject(corev1.ObjectReference{
			Kind: "Pod", Name: "my-app-7d9f8c6b4d-x2kqp", Namespace: "default",
			UID: "9c8b7a6f-5e4d-3c2b-1a09-fedcba987654",
		}),
		kubeeventtest.WithReason("BackOff"),
		kubeeventtest.WithMessage("Back-off restarting failed container app in pod my-app-7d9f8c6b4d-x2kqp"),
		kubeeventtest.WithType("Warning"),
		kubeeventtest.WithTimes(sampleTime, sampleTimeLater),
		kubeeventtest.WithCount(2),
	)

	// An events.k8s.io/v1 event - the scheduler, in this case - first
	// observed and then repeated: repeats accumulate in Series instead of
	// Count/lastTimestamp, which core/v1 has no room for.
	eventsAPIAdded := kubeeventtest.EventsAPIEvent(
		kubeeventtest.WithName("my-app-7d9f8c6b4d-x2kqp.18f2a3b9c0d1e2f4"),
		kubeeventtest.WithNamespace("default"),
		kubeeventtest.WithUID("7a6b5c4d-3e2f-1a0b-9c8d-123456fedcba"),
		kubeeventtest.WithCreationTimestamp(sampleTime),
		kubeeventtest.WithInvolvedObject(corev1.ObjectReference{
			Kind: "Pod", Name: "my-app-7d9f8c6b4d-x2kqp", Namespace: "default",
			UID: "9c8b7a6f-5e4d-3c2b-1a09-fedcba987654",
		}),
		kubeeventtest.WithReason("Scheduled"),
		kubeeventtest.WithMessage("Successfully assigned default/my-app-7d9f8c6b4d-x2kqp to node-1"),
		kubeeventtest.WithEventTime(sampleTime),
	)
	eventsAPIUpdated := kubeeventtest.EventsAPIEvent(
		kubeeventtest.WithName("my-app-7d9f8c6b4d-x2kqp.18f2a3b9c0d1e2f4"),
		kubeeventtest.WithNamespace("default"),
		kubeeventtest.WithUID("7a6b5c4d-3e2f-1a0b-9c8d-123456fedcba"),
		kubeeventtest.WithCreationTimestamp(sampleTime),
		kubeeventtest.WithInvolvedObject(corev1.ObjectReference{
			Kind: "Pod", Name: "my-app-7d9f8c6b4d-x2kqp", Namespace: "default",
			UID: "9c8b7a6f-5e4d-3c2b-1a09-fedcba987654",
		}),
		kubeeventtest.WithReason("Scheduled"),
		kubeeventtest.WithMessage("Successfully assigned default/my-app-7d9f8c6b4d-x2kqp to node-1"),
		kubeeventtest.WithEventTime(sampleTime),
		kubeeventtest.WithSeries(&corev1.EventSeries{
			Count:            2,
			LastObservedTime: metav1.MicroTime{Time: sampleTimeLater},
		}),
	)

	printEvent(sinks.NewEventData(coreAdded, nil))
	printEvent(sinks.NewEventData(coreUpdated, coreAdded))
	printEvent(sinks.NewEventData(eventsAPIAdded, nil))
	printEvent(sinks.NewEventData(eventsAPIUpdated, eventsAPIAdded))
}

// printEvent marshals an EventData exactly as sinks/stdoutsink.go does and
// writes it as one line.
func printEvent(eData sinks.EventData) {
	line, err := json.Marshal(eData)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal event:", err)
		os.Exit(1)
	}
	fmt.Println(string(line))
}
