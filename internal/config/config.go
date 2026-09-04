/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config loads eventrouter's configuration file and environment
// overrides, installs the logger they select, and builds the Kubernetes
// client the rest of the program runs against.
package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kuoss/eventrouter/internal/logging"
	"github.com/kuoss/eventrouter/sinks"
	"gopkg.in/yaml.v3"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Load parses the input and config file and returns a clientset. It also
// installs the logger the config selects, as a side effect of needing the
// config to know what to install.
type Config struct {
	Kubeconfig       string `yaml:"kubeconfig"`
	EnablePrometheus *bool  `yaml:"enable-prometheus"`
	Log              struct {
		Format string `yaml:"format"`
		Level  string `yaml:"level"`
	} `yaml:"log"`
	Sinks []map[string]any `yaml:"sinks"`
}

var loaded Config

// Load reads configFile - or, if configFile is empty, tries
// /etc/eventrouter/config.yaml (a mounted ConfigMap) and then ./config.yaml
// (a local run) in that order - and returns a clientset.
func Load(configFile string) (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	loaded = Config{EnablePrometheus: boolPtr(true), Sinks: []map[string]any{{"type": "stdout"}}}
	loaded.Log.Format, loaded.Log.Level = "json", "info"

	// Every key above already has a default, so when the caller leaves the
	// path to us, a missing file just means every key keeps its default -
	// see config.example.yaml for all of them, commented. A caller-specified
	// path (main.go's --config flag) is a deliberate choice, though, and a
	// missing file there is a real mistake, not an optional override, so it
	// fails startup instead of silently falling back to defaults. Either
	// way, a file that exists but fails to parse also fails startup.
	candidates := []string{"/etc/eventrouter/config.yaml", "./config.yaml"}
	explicit := configFile != ""
	if explicit {
		candidates = []string{configFile}
	}
	var found bool
	for _, c := range candidates {
		f, openErr := os.Open(filepath.Clean(c))
		if openErr != nil {
			if explicit {
				return nil, fmt.Errorf("open config file %q: %w", c, openErr)
			}
			continue
		}
		found = true
		decodeErr := yaml.NewDecoder(f).Decode(&loaded)
		_ = f.Close()
		if decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
			return nil, fmt.Errorf("ReadInConfig err: %w", decodeErr)
		}
		break
	}
	if !found {
		slog.Info("no config file found, using defaults")
	}
	logging.Setup(loaded.Log.Format, loaded.Log.Level)
	sinks.Configure(loaded.Sinks)

	kubeconfig := loaded.Kubeconfig
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if len(kubeconfig) > 0 {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("BuildConfigFromFlags err: %w", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("InClusterConfig err: %w", err)
		}
	}

	// creates the clientset from kubeconfig
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("NewForConfig err: %w", err)
	}
	return clientset, nil
}

// PrometheusEnabled reports whether the "enable-prometheus" config key is
// set, controlling both the /metrics endpoint and the event counters.
func PrometheusEnabled() bool {
	return loaded.EnablePrometheus != nil && *loaded.EnablePrometheus
}

// SetPrometheusForTest is used by package tests that exercise router behavior.
func SetPrometheusForTest(v bool) { loaded.EnablePrometheus = boolPtr(v) }

func boolPtr(v bool) *bool { return &v }
