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
	"log/slog"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
)

// EventSinkInterface is the interface used to shunt events
type EventSinkInterface interface {
	UpdateEvents(eNew *v1.Event, eOld *v1.Event)
}

// ManufactureSinks builds every sink named in the "sinks" config key: a
// list, each entry a "type" plus that sink's own settings inline (e.g.
// "- type: http, url: ..."). Settings are never shared between entries, so
// the same type can appear more than once with different settings - two
// different HTTP endpoints, say. See config.example.yaml.
func ManufactureSinks() []EventSinkInterface {
	entries := sinkEntries()
	out := make([]EventSinkInterface, 0, len(entries))
	for _, entry := range entries {
		name, _ := entry["type"].(string)
		if name == "" {
			panic(`sinks entry missing required "type"`)
		}
		out = append(out, manufactureSink(name, entrySettings(entry)))
	}
	return out
}

// sinkEntries reads the "sinks" list into the maps each entry decoded as.
// Anything that isn't a list of maps is treated as no entries rather than an
// error. YAML always decodes a list of maps as []interface{} of
// map[string]interface{}; the []map[string]interface{} case exists for
// viper.Set (as tests, and config's own default, use).
func sinkEntries() []map[string]interface{} {
	switch raw := viper.Get("sinks").(type) {
	case []interface{}:
		entries := make([]map[string]interface{}, 0, len(raw))
		for _, r := range raw {
			if m, ok := r.(map[string]interface{}); ok {
				entries = append(entries, m)
			}
		}
		return entries
	case []map[string]interface{}:
		return raw
	default:
		return nil
	}
}

// entrySettings returns settings scoped to one "sinks" list entry, with
// "type" excluded since it names the sink rather than configuring it.
func entrySettings(entry map[string]interface{}) *viper.Viper {
	v := viper.New()
	for k, val := range entry {
		if k != "type" {
			v.Set(k, val)
		}
	}
	return v
}

// manufactureSink builds the sink named name, reading its settings (if it
// has any) from settings - one "sinks" list entry, with "type" excluded.
func manufactureSink(name string, settings *viper.Viper) EventSinkInterface {
	slog.Info("sink selected", "sink", name)
	switch name {
	case "stdout":
		return NewStdoutSink()
	case "http":
		url := settings.GetString("url")
		if url == "" {
			panic("http sink specified but no url")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		settings.SetDefault("bufferSize", 1500)
		settings.SetDefault("discardMessages", true)

		bufferSize := settings.GetInt("bufferSize")
		overflow := settings.GetBool("discardMessages")

		h := NewHTTPSink(url, overflow, bufferSize)
		go h.Run(make(chan bool))
		return h
	case "s3sink":
		accessKeyID := settings.GetString("accessKeyId")
		if accessKeyID == "" {
			panic("s3sink specified but accessKeyId not specified")
		}

		secretAccessKey := settings.GetString("secretAccessKey")
		if secretAccessKey == "" {
			panic("s3sink specified but secretAccessKey not specified")
		}

		region := settings.GetString("region")
		if region == "" {
			panic("s3sink specified but region not specified")
		}

		bucket := settings.GetString("bucket")
		if bucket == "" {
			panic("s3sink specified but bucket not specified")
		}

		bucketDir := settings.GetString("bucketDir")
		if bucketDir == "" {
			panic("s3sink specified but bucketDir not specified")
		}

		// By default the json is pushed to s3 in not flatenned rfc5424 write format
		// The option to write to s3 is in the flattened json format which will help in
		// using the data in redshift with least effort
		settings.SetDefault("outputFormat", "rfc5424")
		outputFormat := settings.GetString("outputFormat")
		if outputFormat != "rfc5424" && outputFormat != "flatjson" {
			panic("s3sink specified, but incorrect outputFormat specified. Supported formats are: rfc5424 (default) and flatjson")
		}

		// By default we buffer up to 1500 events, and drop messages if more than
		// 1500 have come in without getting consumed
		settings.SetDefault("bufferSize", 1500)
		settings.SetDefault("discardMessages", true)

		settings.SetDefault("uploadInterval", 120)
		uploadInterval := settings.GetInt("uploadInterval")

		bufferSize := settings.GetInt("bufferSize")
		overflow := settings.GetBool("discardMessages")

		s, err := NewS3Sink(accessKeyID, secretAccessKey, region, bucket, bucketDir, uploadInterval, overflow, bufferSize, outputFormat)
		if err != nil {
			panic(err.Error())
		}

		go s.Run(make(chan bool))
		return s
	case "influxdb":
		host := settings.GetString("host")
		if host == "" {
			panic("influxdb sink specified but host not specified")
		}

		username := settings.GetString("username")
		if username == "" {
			panic("influxdb sink specified but username not specified")
		}

		password := settings.GetString("password")
		if password == "" {
			panic("influxdb sink specified but password not specified")
		}

		settings.SetDefault("name", "k8s")
		settings.SetDefault("secure", false)
		settings.SetDefault("withFields", false)
		settings.SetDefault("insecureSsl", false)
		settings.SetDefault("retentionPolicy", "0")
		settings.SetDefault("clusterName", "default")
		settings.SetDefault("disableCounterMetrics", false)
		settings.SetDefault("concurrency", 1)

		dbName := settings.GetString("name")
		secure := settings.GetBool("secure")
		withFields := settings.GetBool("withFields")
		insecureSsl := settings.GetBool("insecureSsl")
		retentionPolicy := settings.GetString("retentionPolicy")
		cluterName := settings.GetString("clusterName")
		disableCounterMetrics := settings.GetBool("disableCounterMetrics")
		concurrency := settings.GetInt("concurrency")

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
		path := settings.GetString("path")
		if path == "" {
			panic("filesink specified but path not specified")
		}

		return NewFileSink(FileSinkConfig{
			Path:       path,
			MaxSize:    settings.GetInt("maxSize"),
			MaxBackups: settings.GetInt("maxBackups"),
			MaxAge:     settings.GetInt("maxAge"),
			Compress:   settings.GetBool("compress"),
		})
	default:
		err := errors.New("invalid Sink Specified")
		panic(err.Error())
	}
}
