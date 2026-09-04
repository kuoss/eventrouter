package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// AddConfigPath(".") resolves against the process's working directory, so
	// point it at the repo root, where the example config.json lives.
	t.Chdir("../..")

	k8s, err := Load()
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}
