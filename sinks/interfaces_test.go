package sinks

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestManufactureSinks(t *testing.T) {
	t.Run("StdoutSink", func(t *testing.T) {
		viper.Set("sink", "stdout")
		got := ManufactureSinks()
		require.Len(t, got, 1)
		_, ok := got[0].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink")
	})

	t.Run("HTTPSink", func(t *testing.T) {
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

	t.Run("MultipleSinks", func(t *testing.T) {
		viper.Set("sink", []string{"stdout", "http"})
		viper.Set("http.url", "http://localhost")

		got := ManufactureSinks()
		require.Len(t, got, 2)
		_, ok := got[0].(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink first")
		_, ok = got[1].(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink second")
	})

	t.Run("InvalidSink", func(t *testing.T) {
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
