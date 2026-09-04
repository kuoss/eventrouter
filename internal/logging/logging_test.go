package logging

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
	}
	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			require.Equal(t, tc.want, parseLevel(tc.level))
		})
	}
}

func TestSetupLogging(t *testing.T) {
	t.Cleanup(func() { Setup("json", "info") })

	Setup("text", "debug")
	require.Equal(t, slog.LevelDebug, logLevel.Level())
	require.True(t, slog.Default().Enabled(t.Context(), slog.LevelDebug))

	Setup("json", "error")
	require.Equal(t, slog.LevelError, logLevel.Level())
	require.False(t, slog.Default().Enabled(t.Context(), slog.LevelWarn))
}
