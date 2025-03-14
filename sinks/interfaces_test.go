package sinks

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestManufactureSink(t *testing.T) {
	t.Run("GlogSink", func(t *testing.T) {
		viper.Set("sink", "glog")
		sink := ManufactureSink()
		assert.NotNil(t, sink)
		_, ok := sink.(*GlogSink)
		assert.True(t, ok, "Expected GlogSink")
	})

	t.Run("StdoutSink", func(t *testing.T) {
		viper.Set("sink", "stdout")
		viper.Set("stdoutJSONNamespace", "testnamespace")
		sink := ManufactureSink()
		assert.NotNil(t, sink)
		stdoutSink, ok := sink.(*StdoutSink)
		assert.True(t, ok, "Expected StdoutSink")
		assert.Equal(t, "testnamespace", stdoutSink.namespace) // correct field access
	})

	t.Run("HTTPSink", func(t *testing.T) {
		viper.Set("sink", "http")
		viper.Set("httpSinkUrl", "http://localhost")
		viper.Set("httpSinkBufferSize", 1500)
		viper.Set("httpSinkDiscardMessages", true)

		sink := ManufactureSink()
		assert.NotNil(t, sink)
		httpSink, ok := sink.(*HTTPSink)
		assert.True(t, ok, "Expected HTTPSink")

		// Check if there's a method or public field to access the URL
		// Assuming url is a public field in HTTPSink struct
		assert.Equal(t, "http://localhost", httpSink.SinkURL)
	})

	t.Run("InvalidSink", func(t *testing.T) {
		viper.Set("sink", "invalid")

		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, r, "invalid Sink Specified")
			}
		}()

		ManufactureSink()
	})

	// Additional tests for each sink type can be added below
}
