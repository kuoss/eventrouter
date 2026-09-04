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
		viper.Set("sink", "stdout")
		got := ManufactureSinks()
		require.Len(t, got, 1)
		_, ok := got[0].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink")
	})

	t.Run("HTTPSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sink", "http")
		viper.Set("http.url", "http://localhost")
		viper.Set("http.bufferSize", 1500)
		viper.Set("http.discardMessages", true)

		got := ManufactureSinks()
		require.Len(t, got, 1)
		httpSink, ok := got[0].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink")

		require.Equal(t, "http://localhost", httpSink.SinkURL)
	})

	t.Run("FileSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sink", "filesink")
		viper.Set("filesink.path", filepath.Join(t.TempDir(), "events.log"))

		got := ManufactureSinks()
		require.Len(t, got, 1)
		_, ok := got[0].(*FileSink)
		require.True(t, ok, "Expected FileSink")
	})

	t.Run("SinksList", func(t *testing.T) {
		viper.Reset()
		// "sinks" is a list of {type, ...settings}, each entry's settings
		// inline rather than shared - this is what lets two entries name the
		// same type with different settings, unlike "sink".
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

	t.Run("SinkAndSinksTogether", func(t *testing.T) {
		viper.Reset()
		viper.Set("sink", "stdout")
		viper.Set("sinks", []map[string]interface{}{
			{"type": "http", "url": "http://a"},
		})

		got := ManufactureSinks()
		require.Len(t, got, 2)
		_, ok := got[0].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink from \"sink\" first")
		_, ok = got[1].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink from \"sinks\" second")
	})

	t.Run("SinksEntryMissingType", func(t *testing.T) {
		viper.Reset()
		viper.Set("sinks", []map[string]interface{}{
			{"url": "http://a"},
		})

		require.PanicsWithValue(t, `sinks entry missing required "type"`, func() {
			ManufactureSinks()
		})
	})

	t.Run("InvalidSink", func(t *testing.T) {
		viper.Reset()
		viper.Set("sink", "invalid")

		defer func() {
			if r := recover(); r != nil {
				require.Contains(t, r, "invalid Sink Specified")
			}
		}()

		ManufactureSinks()
	})

	// Additional tests for each sink type can be added below
}
