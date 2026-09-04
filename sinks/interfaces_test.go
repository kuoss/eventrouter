package sinks

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestManufactureSink(t *testing.T) {
	t.Run("StdoutSink", func(t *testing.T) {
		viper.Set("sink", "stdout")
		sink := ManufactureSink()
		require.NotNil(t, sink)
		_, ok := sink.(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink")
	})

	t.Run("HTTPSink", func(t *testing.T) {
		viper.Set("sink", "http")
		viper.Set("http.url", "http://localhost")
		viper.Set("http.bufferSize", 1500)
		viper.Set("http.discardMessages", true)

		sink := ManufactureSink()
		require.NotNil(t, sink)
		httpSink, ok := sink.(*HTTPSink)
		require.True(t, ok, "Expected HTTPSink")

		// Check if there's a method or public field to access the URL
		// Assuming url is a public field in HTTPSink struct
		require.Equal(t, "http://localhost", httpSink.SinkURL)
	})

	t.Run("InvalidSink", func(t *testing.T) {
		viper.Set("sink", "invalid")

		defer func() {
			if r := recover(); r != nil {
				require.Contains(t, r, "invalid Sink Specified")
			}
		}()

		ManufactureSink()
	})

	// Additional tests for each sink type can be added below
}
