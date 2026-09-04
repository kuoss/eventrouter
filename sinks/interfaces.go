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

package sinks

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
)

// EventSinkInterface is the interface used to shunt events
type EventSinkInterface interface {
	UpdateEvents(eNew *v1.Event, eOld *v1.Event)
}

// ManufactureSinks builds every sink named by the "sink" config key - a
// single name (`sink: stdout`) or a list of them (`sink: [stdout, http]`) -
// each reading its own settings (if it has any) from the nested block of the
// same name. See config.example.yaml.
func ManufactureSinks() []EventSinkInterface {
	names := sinkNames()
	out := make([]EventSinkInterface, 0, len(names))
	for _, name := range names {
		out = append(out, manufactureSink(name))
	}
	return out
}

// sinkNames normalizes the "sink" config key, which may be given as a single
// scalar or a list, into the list of sink names to build.
func sinkNames() []string {
	switch v := viper.Get("sink").(type) {
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, s := range v {
			names = append(names, fmt.Sprint(s))
		}
		return names
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}

func manufactureSink(name string) EventSinkInterface {
	slog.Info("sink selected", "sink", name)
	switch name {
	case "stdout":
		return NewStdoutSink()
	case "http":
		url := viper.GetString("http.url")
		if url == "" {
			panic("http sink specified but no http.url")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		viper.SetDefault("http.bufferSize", 1500)
		viper.SetDefault("http.discardMessages", true)

		bufferSize := viper.GetInt("http.bufferSize")
		overflow := viper.GetBool("http.discardMessages")

		h := NewHTTPSink(url, overflow, bufferSize)
		go h.Run(make(chan bool))
		return h
	case "s3sink":
		accessKeyID := viper.GetString("s3sink.accessKeyId")
		if accessKeyID == "" {
			panic("s3sink specified but s3sink.accessKeyId not specified")
		}

		secretAccessKey := viper.GetString("s3sink.secretAccessKey")
		if secretAccessKey == "" {
			panic("s3sink specified but s3sink.secretAccessKey not specified")
		}

		region := viper.GetString("s3sink.region")
		if region == "" {
			panic("s3sink specified but s3sink.region not specified")
		}

		bucket := viper.GetString("s3sink.bucket")
		if bucket == "" {
			panic("s3sink specified but s3sink.bucket not specified")
		}

		bucketDir := viper.GetString("s3sink.bucketDir")
		if bucketDir == "" {
			panic("s3sink specified but s3sink.bucketDir not specified")
		}

		// By default the json is pushed to s3 in not flatenned rfc5424 write format
		// The option to write to s3 is in the flattened json format which will help in
		// using the data in redshift with least effort
		viper.SetDefault("s3sink.outputFormat", "rfc5424")
		outputFormat := viper.GetString("s3sink.outputFormat")
		if outputFormat != "rfc5424" && outputFormat != "flatjson" {
			panic("s3sink specified, but incorrect s3sink.outputFormat specified. Supported formats are: rfc5424 (default) and flatjson")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		viper.SetDefault("s3sink.bufferSize", 1500)
		viper.SetDefault("s3sink.discardMessages", true)

		viper.SetDefault("s3sink.uploadInterval", 120)
		uploadInterval := viper.GetInt("s3sink.uploadInterval")

		bufferSize := viper.GetInt("s3sink.bufferSize")
		overflow := viper.GetBool("s3sink.discardMessages")

		s, err := NewS3Sink(accessKeyID, secretAccessKey, region, bucket, bucketDir, uploadInterval, overflow, bufferSize, outputFormat)
		if err != nil {
			panic(err.Error())
		}

		go s.Run(make(chan bool))
		return s
	case "influxdb":
		host := viper.GetString("influxdb.host")
		if host == "" {
			panic("influxdb sink specified but influxdb.host not specified")
		}

		username := viper.GetString("influxdb.username")
		if username == "" {
			panic("influxdb sink specified but influxdb.username not specified")
		}

		password := viper.GetString("influxdb.password")
		if password == "" {
			panic("influxdb sink specified but influxdb.password not specified")
		}

		viper.SetDefault("influxdb.name", "k8s")
		viper.SetDefault("influxdb.secure", false)
		viper.SetDefault("influxdb.withFields", false)
		viper.SetDefault("influxdb.insecureSsl", false)
		viper.SetDefault("influxdb.retentionPolicy", "0")
		viper.SetDefault("influxdb.clusterName", "default")
		viper.SetDefault("influxdb.disableCounterMetrics", false)
		viper.SetDefault("influxdb.concurrency", 1)

		dbName := viper.GetString("influxdb.name")
		secure := viper.GetBool("influxdb.secure")
		withFields := viper.GetBool("influxdb.withFields")
		insecureSsl := viper.GetBool("influxdb.insecureSsl")
		retentionPolicy := viper.GetString("influxdb.retentionPolicy")
		cluterName := viper.GetString("influxdb.clusterName")
		disableCounterMetrics := viper.GetBool("influxdb.disableCounterMetrics")
		concurrency := viper.GetInt("influxdb.concurrency")

		cfg := InfluxdbConfig{
			User:                  username,
			Password:              password,
			Secure:                secure,
			Host:                  host,
			DbName:                dbName,
			WithFields:            withFields,
			InsecureSsl:           insecureSsl,
			RetentionPolicy:       retentionPolicy,
			ClusterName:           cluterName,
			DisableCounterMetrics: disableCounterMetrics,
			Concurrency:           concurrency,
		}

		influx, err := NewInfluxdbSink(cfg)
		if err != nil {
			panic(err.Error())
		}
		return influx
	case "filesink":
		path := viper.GetString("filesink.path")
		if path == "" {
			panic("filesink specified but filesink.path not specified")
		}

		return NewFileSink(FileSinkConfig{
			Path:       path,
			MaxSize:    viper.GetInt("filesink.maxSize"),
			MaxBackups: viper.GetInt("filesink.maxBackups"),
			MaxAge:     viper.GetInt("filesink.maxAge"),
			Compress:   viper.GetBool("filesink.compress"),
		})
	default:
		err := errors.New("invalid Sink Specified")
		panic(err.Error())
	}
}
