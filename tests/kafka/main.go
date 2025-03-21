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

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kuoss/eventrouter/sinks"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ref "k8s.io/client-go/tools/reference"
)

type KafkaEnv struct {
	Brokers  []string
	Topic    string
	Async    bool
	RetryMax int
}

func loadConfig() KafkaEnv {
	viper.SetDefault("ASYNC", true)
	viper.SetDefault("RETRYMAX", 5)

	viper.AutomaticEnv() // Read in environment variables

	brokers := viper.GetStringSlice("KAFKA_BROKERS")
	topic := viper.GetString("KAFKA_TOPIC")
	async := viper.GetBool("ASYNC")
	retryMax := viper.GetInt("RETRYMAX")

	if len(brokers) == 0 || topic == "" {
		log.Fatal("Missing required environment variables: KAFKA_BROKERS, KAFKA_TOPIC")
	}

	return KafkaEnv{
		Brokers:  brokers,
		Topic:    topic,
		Async:    async,
		RetryMax: retryMax,
	}
}

func main() {
	k := loadConfig()
	kSink, err := sinks.NewKafkaSink(k.Brokers, k.Topic, k.Async, k.RetryMax, "user", "password")
	if err != nil {
		log.Fatal(err)
	}

	testPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			SelfLink:  "/api/version/pods/somePod",
			Name:      "somePod",
			Namespace: "someNameSpace",
			UID:       "some-UID",
		},
		Spec: v1.PodSpec{},
	}

	podRef, err := ref.GetReference(scheme.Scheme, testPod)
	if err != nil {
		log.Fatal(err)
	}

	kvs := map[string]string{
		"CreateInCluster": "Mock create event on Pod",
		"UpdateInCluster": "Mock update event on Pod",
		"DeleteInCluster": "Mock delete event on Pod",
	}

	var oldData, newData *v1.Event

	for k, v := range kvs {
		newData = newMockEvent(podRef, v1.EventTypeWarning, k, v)
		kSink.UpdateEvents(newData, oldData)
		oldData = newData
		time.Sleep(time.Second)
	}
}

// TODO: This function should be moved where it can be re-used...
func newMockEvent(ref *v1.ObjectReference, eventtype, reason, message string) *v1.Event {
	tm := metav1.Time{
		Time: time.Now(),
	}
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", ref.Name, tm.UnixNano()),
			Namespace: ref.Namespace,
		},
		InvolvedObject: *ref,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: tm,
		LastTimestamp:  tm,
		Count:          1,
		Type:           eventtype,
	}
}
