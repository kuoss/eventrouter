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
	"sync"

	"github.com/kuoss/eventrouter/internal/config"
	"github.com/kuoss/eventrouter/internal/router"
	"github.com/kuoss/eventrouter/internal/shutdown"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/client-go/informers"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// addr tells us what address to have the Prometheus metrics listen on.
var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

// showVersion prints the version and exits, which also makes the image
// runnable as a smoke test.
var showVersion = flag.Bool("version", false, "Print the version and exit.")

// main entry point of the program
func main() {
	var wg sync.WaitGroup

	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	clientset, err := config.Load()
	if err != nil {
		slog.Error("config.Load failed", "err", err)
		os.Exit(1)
	}
	slog.Info("starting eventrouter", "version", version)
	sharedInformers := informers.NewSharedInformerFactory(clientset, config.ResyncInterval())
	eventsInformer := sharedInformers.Core().V1().Events()

	// TODO: Support locking for HA https://github.com/kubernetes/kubernetes/pull/42666
	eventRouter := router.NewEventRouter(clientset, eventsInformer)
	stop := shutdown.Signal()

	// Startup the http listener for Prometheus Metrics endpoint.
	if config.PrometheusEnabled() {
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
