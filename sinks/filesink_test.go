package sinks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestFileSink_UpdateEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.log")
	sink := NewFileSink(FileSinkConfig{Path: path})

	sink.UpdateEvents(&corev1.Event{Reason: "hello"}, nil)
	sink.UpdateEvents(&corev1.Event{Reason: "world"}, &corev1.Event{Reason: "hello"})

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	require.Len(t, lines, 2)

	var added EventData
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &added))
	require.Equal(t, "ADDED", added.Verb)
	require.Equal(t, "hello", added.Event.Reason)

	var updated EventData
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &updated))
	require.Equal(t, "UPDATED", updated.Verb)
	require.Equal(t, "world", updated.Event.Reason)
}
