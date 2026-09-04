package config

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Load leaves state in the global viper instance (the config file it
	// found, defaults, bound env vars), which would otherwise leak between
	// these two tests regardless of run order.
	viper.Reset()

	// AddConfigPath(".") resolves against the process's working directory, so
	// point it at testdata, whose config.yaml sets a kubeconfig path that
	// does not exist - Load should read the file fine and then fail
	// predictably once it tries to build a client from that path.
	t.Chdir("testdata")

	k8s, err := Load()
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}

func TestLoadWithEnvOverride(t *testing.T) {
	viper.Reset()

	// Resolve the fixture before Chdir, since a relative path would
	// otherwise resolve against the directory Load searches from below.
	fixture, err := filepath.Abs("testdata/config.yaml")
	require.NoError(t, err)
	t.Setenv("EVENTROUTER_CONFIG", fixture)

	// An empty directory: only EVENTROUTER_CONFIG, not AddConfigPath("."),
	// can be what finds the fixture here.
	t.Chdir(t.TempDir())

	k8s, err := Load()
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}

func TestLoadWithoutConfigFile(t *testing.T) {
	viper.Reset()

	// An empty directory: no config.yaml for AddConfigPath(".") to find.
	t.Chdir(t.TempDir())

	// Force the not-in-cluster branch regardless of where the test happens to
	// run, so the assertion below doesn't depend on the environment.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	// A missing config file is not an error - every key already has a
	// default - so Load should get all the way to InClusterConfig and fail
	// there instead of bailing out earlier on ReadInConfig.
	k8s, err := Load()
	require.EqualError(t, err, "InClusterConfig err: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	require.Nil(t, k8s)
}
