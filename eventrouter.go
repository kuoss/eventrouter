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
	"fmt"
	"log/slog"

	"github.com/kuoss/eventrouter/internal/kubeevent"
	"github.com/kuoss/eventrouter/sinks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// eventCounterLabels are the labels every event counter carries. "source" is
// the reporting node and is empty when the event does not name one; "component"
// is the reporting controller.
var eventCounterLabels = []string{
	"involved_object_kind",
	"involved_object_name",
	"involved_object_namespace",
	"reason",
	"source",
	"component",
}

var (
	kubernetesWarningEventCounterVec = newEventCounterVec(
		"eventrouter_warnings_total", "Total number of warning events in the kubernetes cluster")
	kubernetesNormalEventCounterVec = newEventCounterVec(
		"eventrouter_normal_total", "Total number of normal events in the kubernetes cluster")
	kubernetesInfoEventCounterVec = newEventCounterVec(
		"eventrouter_info_total", "Total number of info events in the kubernetes cluster")
	kubernetesUnknownEventCounterVec = newEventCounterVec(
		"eventrouter_unknown_total", "Total number of events of unknown type in the kubernetes cluster")
)

func newEventCounterVec(name, help string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, eventCounterLabels)
}

// EventRouter is responsible for maintaining a stream of kubernetes
// system Events and pushing them to another channel for storage
type EventRouter struct {
	// kubeclient is the main kubernetes interface
	kubeClient kubernetes.Interface

	// store of events populated by the shared informer
	eLister corelisters.EventLister

	// returns true if the event store has been synced
	eListerSynched cache.InformerSynced

	// event sink
	// TODO: Determine if we want to support multiple sinks.
	eSink sinks.EventSinkInterface
}

// NewEventRouter will create a new event router using the input params
func NewEventRouter(kubeClient kubernetes.Interface, eventsInformer coreinformers.EventInformer) *EventRouter {
	if viper.GetBool("enable-prometheus") {
		prometheus.MustRegister(kubernetesWarningEventCounterVec)
		prometheus.MustRegister(kubernetesNormalEventCounterVec)
		prometheus.MustRegister(kubernetesInfoEventCounterVec)
		prometheus.MustRegister(kubernetesUnknownEventCounterVec)
	}

	er := &EventRouter{
		kubeClient: kubeClient,
		eSink:      sinks.ManufactureSink(),
	}
	_, err := eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    er.addEvent,
		UpdateFunc: er.updateEvent,
		DeleteFunc: er.deleteEvent,
	})
	if err != nil {
		slog.Error("AddEventHandler failed", "err", err)
	}
	er.eLister = eventsInformer.Lister()
	er.eListerSynched = eventsInformer.Informer().HasSynced
	return er
}

// Run starts the EventRouter/Controller.
func (er *EventRouter) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer slog.Info("shutting down EventRouter")

	slog.Info("starting EventRouter")

	// here is where we kick the caches into gear
	if !cache.WaitForCacheSync(stopCh, er.eListerSynched) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	<-stopCh
}

// addEvent is called when an event is created, or during the initial list
func (er *EventRouter) addEvent(obj interface{}) {
	e := obj.(*v1.Event)
	prometheusEvent(e)
	er.eSink.UpdateEvents(e, nil)
}

// updateEvent is called any time there is an update to an existing event
func (er *EventRouter) updateEvent(objOld interface{}, objNew interface{}) {
	eOld := objOld.(*v1.Event)
	eNew := objNew.(*v1.Event)
	prometheusEvent(eNew)
	er.eSink.UpdateEvents(eNew, eOld)
}

// prometheusEvent is called when an event is added or updated
func prometheusEvent(event *v1.Event) {
	if !viper.GetBool("enable-prometheus") {
		return
	}

	var counterVec *prometheus.CounterVec
	switch event.Type {
	case "Normal":
		counterVec = kubernetesNormalEventCounterVec
	case "Warning":
		counterVec = kubernetesWarningEventCounterVec
	case "Info":
		counterVec = kubernetesInfoEventCounterVec
	default:
		counterVec = kubernetesUnknownEventCounterVec
	}

	// Read through kubeevent rather than off the event: an event written
	// through events.k8s.io/v1 carries no source at all.
	counter, err := counterVec.GetMetricWithLabelValues(
		event.InvolvedObject.Kind,
		event.InvolvedObject.Name,
		event.InvolvedObject.Namespace,
		event.Reason,
		kubeevent.Host(event),
		kubeevent.Component(event),
	)
	if err != nil {
		slog.Warn("could not get event counter", "err", err)
		return
	}
	counter.Add(1)
}

// deleteEvent should only occur when the system garbage collects events via TTL expiration
func (er *EventRouter) deleteEvent(obj interface{}) {
	e, err := toEventPointer(obj)
	if err != nil {
		slog.Warn("toEventPointer failed", "err", err)
		return
	}
	slog.Debug("event deleted from the system", "event", e)
}

func toEventPointer(obj interface{}) (*v1.Event, error) {
	e, ok := obj.(*v1.Event)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}
	return e, nil
}
