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

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kuoss/eventrouter/internal/logging"
	"github.com/kuoss/eventrouter/internal/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// addr tells us what address to have the Prometheus metrics listen on.
var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

// showVersion prints the version and exits, which also makes the image
// runnable as a smoke test.
var showVersion = flag.Bool("version", false, "Print the version and exit.")

// setup a signal hander to gracefully exit
func sigHandler() <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGTERM, // Termination Request
			syscall.SIGSEGV, // FullDerp
			syscall.SIGABRT, // Abnormal termination
			syscall.SIGILL,  // illegal instruction
			syscall.SIGFPE)  // floating point - this is why we can't have nice things
		sig := <-c
		slog.Warn("signal detected, shutting down", "signal", sig)
		close(stop)
	}()
	return stop
}

// loadConfig will parse input + config file and return a clientset
func loadConfig() (kubernetes.Interface, error) {
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
	viper.SetDefault("resync-interval", time.Minute*30)
	viper.SetDefault("enable-prometheus", true)
	viper.SetDefault("log-format", "json")
	viper.SetDefault("log-level", "info")

	err = viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("ReadInConfig err: %w", err)
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

// main entry point of the program
func main() {
	var wg sync.WaitGroup

	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	clientset, err := loadConfig()
	if err != nil {
		slog.Error("loadConfig failed", "err", err)
		os.Exit(1)
	}
	slog.Info("starting eventrouter", "version", version)
	sharedInformers := informers.NewSharedInformerFactory(clientset, viper.GetDuration("resync-interval"))
	eventsInformer := sharedInformers.Core().V1().Events()

	// TODO: Support locking for HA https://github.com/kubernetes/kubernetes/pull/42666
	eventRouter := router.NewEventRouter(clientset, eventsInformer)
	stop := sigHandler()

	// Startup the http listener for Prometheus Metrics endpoint.
	if viper.GetBool("enable-prometheus") {
		go func() {
			slog.Info("starting prometheus metrics", "address", *addr)
			http.Handle("/metrics", promhttp.Handler())
			slog.Error("prometheus metrics listener stopped", "err", http.ListenAndServe(*addr, nil))
		}()
	}

	// Startup the EventRouter
	wg.Add(1)
	go func() {
		defer wg.Done()
		eventRouter.Run(stop)
	}()

	// Startup the Informer(s)
	slog.Info("starting shared informer(s)")
	sharedInformers.Start(stop)
	wg.Wait()
	slog.Warn("exiting main()")
	os.Exit(1)
}
