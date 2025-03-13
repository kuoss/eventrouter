/*
Copyright 2017 The Contributors

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

package sinks

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	v1 "k8s.io/api/core/v1"
)

var (
	LabelPodId = LabelDescriptor{
		Key:         "pod_id",
		Description: "The unique ID of the pod",
	}

	LabelPodName = LabelDescriptor{
		Key:         "pod_name",
		Description: "The name of the pod",
	}

	LabelNamespaceName = LabelDescriptor{
		Key:         "namespace_name",
		Description: "The name of the namespace",
	}

	LabelHostname = LabelDescriptor{
		Key:         "hostname",
		Description: "Hostname where the container ran",
	}
)

const (
	eventMeasurementName = "k8s_events"
	// Event special tags
	eventUID = "uid"
	// Value Field name
	valueField = "value"
	// Event special tags
	dbNotFoundError = "database not found"
)

type LabelDescriptor struct {
	// Key to use for the label.
	Key string `json:"key,omitempty"`

	// Description of the label.
	Description string `json:"description,omitempty"`
}

type InfluxDBSink struct {
	config InfluxdbConfig
	client influxdb2.Client
	sync.RWMutex
	dbExists bool
}

type InfluxdbConfig struct {
	User                  string
	Password              string
	Secure                bool
	Host                  string
	DbName                string
	WithFields            bool
	InsecureSsl           bool
	RetentionPolicy       string
	ClusterName           string
	DisableCounterMetrics bool
	Concurrency           int
}

// Returns a thread-safe implementation of EventSinkInterface for InfluxDB.
func NewInfluxdbSink(cfg InfluxdbConfig) (EventSinkInterface, error) {
	protocol := "http"
	if cfg.Secure {
		protocol = "https"
	}

	serverURL := fmt.Sprintf("%s://%s", protocol, cfg.Host)
	authToken := fmt.Sprintf("%s:%s", cfg.User, cfg.Password)
	client := influxdb2.NewClientWithOptions(serverURL, authToken,
		influxdb2.DefaultOptions().SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.InsecureSsl}),
	)

	return &InfluxDBSink{
		config: cfg,
		client: client,
	}, nil
}

func (sink *InfluxDBSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	sink.Lock()
	defer sink.Unlock()

	var point *write.Point
	var err error
	if sink.config.WithFields {
		point, err = eventToPointWithFields(eNew)
	} else {
		point, err = eventToPoint(eNew)
	}
	if err != nil {
		glog.Warningf("Failed to convert event to point: %v", err)
	}

	point.AddTag("cluster_name", sink.config.ClusterName)
	sink.sendData([]*write.Point{point})
}

func getEventValue(event *v1.Event) (string, error) {
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func eventToPointWithFields(event *v1.Event) (*write.Point, error) {
	tags := map[string]string{
		eventUID:               string(event.UID),
		"message":              event.Message,
		"object_name":          event.InvolvedObject.Name,
		"type":                 event.Type,
		"kind":                 event.InvolvedObject.Kind,
		"component":            event.Source.Component,
		"reason":               event.Reason,
		LabelNamespaceName.Key: event.Namespace,
		LabelHostname.Key:      event.Source.Host,
	}
	if event.InvolvedObject.Kind == "Pod" {
		tags[LabelPodId.Key] = string(event.InvolvedObject.UID)
	}
	fields := map[string]interface{}{}
	ts := event.LastTimestamp.Time.UTC()
	return influxdb2.NewPoint("events", tags, fields, ts), nil
}

func eventToPoint(event *v1.Event) (*write.Point, error) {
	value, err := getEventValue(event)
	if err != nil {
		return nil, err
	}

	tags := map[string]string{
		eventUID: string(event.UID),
	}
	if event.InvolvedObject.Kind == "Pod" {
		tags[LabelPodId.Key] = string(event.InvolvedObject.UID)
		tags[LabelPodName.Key] = event.InvolvedObject.Name
	}
	tags[LabelHostname.Key] = event.Source.Host

	fields := map[string]interface{}{
		valueField: value,
	}

	ts := event.LastTimestamp.Time.UTC()
	point := influxdb2.NewPoint(eventMeasurementName, tags, fields, ts)
	return point, nil
}

func (sink *InfluxDBSink) sendData(points []*write.Point) {
	writeAPI := sink.client.WriteAPIBlocking("", sink.config.DbName)

	// Attempt to write the points
	if err := writeAPI.WritePoint(context.Background(), points...); err != nil {
		glog.Errorf("InfluxDB write failed: %v", err)
		// Handle potential connection issues
		if strings.Contains(err.Error(), dbNotFoundError) {
			sink.dbExists = false
		}
	}
}
