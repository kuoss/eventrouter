package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	k8s, err := loadConfig()
	require.EqualError(t, err, `ReadInConfig err: While parsing config: invalid character '"' after object key:value pair`)
	require.Nil(t, k8s)
}
