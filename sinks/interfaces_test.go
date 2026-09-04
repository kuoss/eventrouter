package sinks

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestManufactureSinks(t *testing.T) {
	t.Run("StdoutSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{{"type": "stdout"}})

		got := ManufactureSinks()
		require.Len(t, got, 1)
		_, ok := got[0].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink")
	})

	t.Run("HTTPSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{
			{"type": "http", "url": "http://localhost", "bufferSize": 1500, "discardMessages": true},
		})

		got := ManufactureSinks()
		require.Len(t, got, 1)
		httpSink, ok := got[0].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink")

		require.Equal(t, "http://localhost", httpSink.SinkURL)
	})

	t.Run("FileSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{
			{"type": "filesink", "path": filepath.Join(t.TempDir(), "events.log")},
		})

		got := ManufactureSinks()
		require.Len(t, got, 1)
		_, ok := got[0].(*FileSink)
		require.True(t, ok, "Expected FileSink")
	})

	t.Run("MultipleEntriesSameType", func(t *testing.T) {
		viper.Reset()
		// Two entries of the same type, each with its own settings - the
		// reason "sinks" is a list of {type, ...} rather than a top-level
		// block per type: one block could never express this.
		viper.Set("sinks", []map[string]interface{}{
			{"type": "http", "url": "http://a"},
			{"type": "http", "url": "http://b"},
			{"type": "stdout"},
		})

		got := ManufactureSinks()
		require.Len(t, got, 3)
		first, ok := got[0].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink first")
		require.Equal(t, "http://a", first.SinkURL)
		second, ok := got[1].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink second")
		require.Equal(t, "http://b", second.SinkURL)
		_, ok = got[2].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink third")
	})

	t.Run("EntryMissingType", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{
			{"url": "http://a"},
		})

		require.PanicsWithValue(t, `sinks entry missing required "type"`, func() {
			ManufactureSinks()
		})
	})

	t.Run("InvalidType", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{{"type": "invalid"}})

		defer func() {
			if r := recover(); r != nil {
				require.Contains(t, r, "invalid Sink Specified")
			}
		}()

		ManufactureSinks()
	})

	t.Run("NoSinksConfigured", func(t *testing.T) {
		viper.Reset()
		got := ManufactureSinks()
		require.Empty(t, got)
	})
}
