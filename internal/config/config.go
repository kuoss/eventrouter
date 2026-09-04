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
	"log/slog"
	"os"

	"github.com/kuoss/eventrouter/internal/logging"
	"github.com/spf13/viper"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Load parses the input and config file and returns a clientset. It also
// installs the logger the config selects, as a side effect of needing the
// config to know what to install.
func Load() (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	// leverages a file|(ConfigMap)
	// to be located at /etc/eventrouter/config
	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/eventrouter/")
	viper.AddConfigPath(".")
	viper.SetDefault("kubeconfig", "")
	viper.SetDefault("sink", "stdout")
	viper.SetDefault("enable-prometheus", true)
	viper.SetDefault("log-format", "json")
	viper.SetDefault("log-level", "info")

	// Every key above already has a default, so a config file is an optional
	// override, not a requirement: a missing one just means every key keeps
	// its default. A file that exists but fails to parse is a real mistake,
	// though, and still fails startup.
	err = viper.ReadInConfig()
	var notFound viper.ConfigFileNotFoundError
	if err != nil && !errors.As(err, &notFound) {
		return nil, fmt.Errorf("ReadInConfig err: %w", err)
	}
	if err != nil {
		slog.Info("no config file found, using defaults and environment overrides")
	}

	err = viper.BindEnv("kubeconfig") // Allows the KUBECONFIG env var to override where the kubeconfig is
	if err != nil {
		return nil, fmt.Errorf("BindEnv err: %w", err)
	}

	// Allows LOG_FORMAT/LOG_LEVEL to override the config file, so verbosity can
	// be changed without editing the ConfigMap.
	err = viper.BindEnv("log-format", "LOG_FORMAT")
	if err != nil {
		return nil, fmt.Errorf("BindEnv err: %w", err)
	}
	err = viper.BindEnv("log-level", "LOG_LEVEL")
	if err != nil {
		return nil, fmt.Errorf("BindEnv err: %w", err)
	}
	logging.Setup(viper.GetString("log-format"), viper.GetString("log-level"))

	// Allow specifying a custom config file via the EVENTROUTER_CONFIG env var
	if forceCfg := os.Getenv("EVENTROUTER_CONFIG"); forceCfg != "" {
		viper.SetConfigFile(forceCfg)
	}
	kubeconfig := viper.GetString("kubeconfig")
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
	return viper.GetBool("enable-prometheus")
}
