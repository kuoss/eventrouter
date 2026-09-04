// Command gen writes tests/sample/pod-log.ndjson: the JSON lines eventrouter's
// own logger emits during startup, interleaved with one JSON line for a
// Kubernetes Event, in the shape `kubectl logs` shows once it merges the
// container's stdout and stderr. Run it with `make sample`; regenerate the
// file whenever a log message wording, or the EventData JSON shape, changes.
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kuoss/eventrouter/sinks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sampleTime stands in for wall-clock time so the generated file is the same
// on every run instead of re-diffing on its timestamps alone.
var sampleTime = time.Date(2026, 9, 4, 6, 35, 20, 0, time.UTC)

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

	// One Kubernetes Event, marshaled exactly as sinks/stdoutsink.go prints it.
	event := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "my-app-7d9f8c6b4d-x2kqp.18f2a3b9c0d1e2f3",
			Namespace:         "default",
			UID:               "3f1e2a4b-1234-4a5b-9c6d-abcdef123456",
			CreationTimestamp: metav1.Time{Time: sampleTime},
		},
		InvolvedObject: v1.ObjectReference{
			Kind:      "Pod",
			Name:      "my-app-7d9f8c6b4d-x2kqp",
			Namespace: "default",
			UID:       "9c8b7a6f-5e4d-3c2b-1a09-fedcba987654",
		},
		Reason:         "Scheduled",
		Message:        "Successfully assigned default/my-app-7d9f8c6b4d-x2kqp to node-1",
		Source:         v1.EventSource{Component: "kubelet", Host: "node-1"},
		FirstTimestamp: metav1.Time{Time: sampleTime},
		LastTimestamp:  metav1.Time{Time: sampleTime},
		Count:          1,
		Type:           "Normal",
	}
	eventLine, err := json.Marshal(sinks.NewEventData(event, nil))
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal event:", err)
		os.Exit(1)
	}
	fmt.Println(string(eventLine))
}
