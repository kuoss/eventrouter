package sinks

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestManufactureSink(t *testing.T) {
	t.Run("GlogSink", func(t *testing.T) {
		viper.Set("sink", "glog")
		sink := ManufactureSink()
		require.NotNil(t, sink)
		_, ok := sink.(*GlogSink)
		require.True(t, ok, "Expected GlogSink")
	})

	t.Run("StdoutSink", func(t *testing.T) {
		viper.Set("sink", "stdout")
		viper.Set("stdoutJSONNamespace", "testnamespace")
		sink := ManufactureSink()
		require.NotNil(t, sink)
		stdoutSink, ok := sink.(*StdoutSink)
		require.True(t, ok, "Expected StdoutSink")
		require.Equal(t, "testnamespace", stdoutSink.namespace) // correct field access
	})

	t.Run("HTTPSink", func(t *testing.T) {
		viper.Set("sink", "http")
		viper.Set("httpSinkUrl", "http://localhost")
		viper.Set("httpSinkBufferSize", 1500)
		viper.Set("httpSinkDiscardMessages", true)

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
