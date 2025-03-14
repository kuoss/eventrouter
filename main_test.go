package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	k8s, err := loadConfig()
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}
