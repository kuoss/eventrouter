package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var viper = testConfig{}

type testConfig struct{}

func (testConfig) Reset() {}

func TestLoad(t *testing.T) {
	// Load leaves state in the loaded global (the config file it found,
	// defaults), which would otherwise leak between these tests regardless
	// of run order.
	viper.Reset()

	// With no explicit path, Load's second candidate is ./config.yaml,
	// resolved against the process's working directory - point it at
	// testdata, whose config.yaml sets a kubeconfig path that does not
	// exist, so Load should read the file fine and then fail predictably
	// once it tries to build a client from that path.
	t.Chdir("testdata")

	k8s, err := Load("")
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}

func TestLoadWithExplicitConfigFile(t *testing.T) {
	viper.Reset()

	// Resolve the fixture before Chdir, since a relative path would
	// otherwise resolve against the directory Load searches from below.
	fixture, err := filepath.Abs("testdata/config.yaml")
	require.NoError(t, err)

	// An empty directory: only the explicit path, not the ./config.yaml
	// fallback, can be what finds the fixture here.
	t.Chdir(t.TempDir())

	k8s, err := Load(fixture)
	require.EqualError(t, err, "BuildConfigFromFlags err: stat /var/run/kubernetes/admin.kubeconfig: no such file or directory")
	require.Nil(t, k8s)
}

func TestLoadWithMissingExplicitConfigFile(t *testing.T) {
	viper.Reset()

	// An explicit path (main.go's --config flag) is a deliberate choice, so
	// a missing file there is a real mistake and fails startup, unlike the
	// "search and fall back to defaults" behavior below.
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	k8s, err := Load(missing)
	require.ErrorContains(t, err, "open config file")
	require.ErrorContains(t, err, "does-not-exist.yaml")
	require.Nil(t, k8s)
}

func TestLoadWithoutConfigFile(t *testing.T) {
	viper.Reset()

	// An empty directory: no config.yaml for the ./config.yaml candidate to
	// find, and no /etc/eventrouter/config.yaml on a test machine either.
	t.Chdir(t.TempDir())

	// Force the not-in-cluster branch regardless of where the test happens to
	// run, so the assertion below doesn't depend on the environment.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	// A missing config file is not an error when the path is left to Load -
	// every key already has a default - so Load should get all the way to
	// InClusterConfig and fail there instead of bailing out earlier.
	k8s, err := Load("")
	require.EqualError(t, err, "InClusterConfig err: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	require.Nil(t, k8s)
}

func TestLoadWithInvalidSinksShape(t *testing.T) {
	// The "sinks" config key decodes into []map[string]any: a scalar, or a
	// list of anything but objects, fails to decode rather than silently
	// becoming no sinks or panicking deep inside the sinks package. See
	// README's Configuration section for the documented shapes.
	for name, yaml := range map[string]string{
		"scalar":           "sinks: stdout\n",
		"list of scalars":  "sinks: [stdout]\n",
		"list of non-maps": "sinks: [[stdout]]\n",
	} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			dir := t.TempDir()
			fixture := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(fixture, []byte(yaml), 0o600))

			k8s, err := Load(fixture)
			require.ErrorContains(t, err, "ReadInConfig err")
			require.Nil(t, k8s)
		})
	}
}
