# Eventrouter

This repository contains a simple event router for the [Kubernetes][kubernetes] project. The event router serves as an active watcher of _event_ resource in the kubernetes system, which takes those events and _pushes_ them to a user specified _sink_.  This is useful for a number of different purposes, but most notably long term behavioral analysis of your 
workloads running on your kubernetes cluster. 

## Goals

This project has several objectives, which include: 

* Persist events for longer period of time to allow for system debugging
* Allows operators to forward events to other system(s) for archiving/ML/introspection/etc. 
* It should be relatively low overhead
* Support for multiple _sinks_ should be configurable

## Non-Goals: 

* This service does not provide a querable extension, that is a responsibility of the 
_sink_
* This service does not serve as a storage layer, that is also the responsibility of the _sink_

## Deployment
```
$ kubectl create -f https://raw.githubusercontent.com/kuoss/eventrouter/main/deploy/deploy.yaml
```

The image is published for `linux/amd64`, `linux/arm64` and `linux/arm/v7`.

### Inspecting the output 
```
$ kubectl logs -f deployment/eventrouter -n kube-system 
``` 

## Configuration

Every config key already has a default, so eventrouter runs with no config
file at all. To change one, mount a ConfigMap with a `config.json` at
`/etc/eventrouter/` (see
[`tests/eventrouter/eventrouter-with-configmap.yaml`][configmap-example] for a
full example, including the RBAC eventrouter needs) - or, for local runs, drop
a `config.json` next to the binary. A malformed file (present but not valid
JSON) still fails startup; a missing one does not.

| config key         | env var      | default    | values                                                                |
| ------------------ | ------------ | ---------- | ---------------------------------------------------------------------|
| `kubeconfig`        | `KUBECONFIG` | *(empty)*  | path to a kubeconfig file; empty uses the in-cluster service account |
| `sink`              | -            | `stdout`   | `stdout`, `http`, `s3`, `influxdb`                                   |
| `resync-interval`   | -            | `30m`      | how often the shared informer resyncs                                |
| `enable-prometheus` | -            | `true`     | exposes `/metrics` and the event counters                            |
| `log-format`        | `LOG_FORMAT` | `json`     | `json`, `text`                                                       |
| `log-level`         | `LOG_LEVEL`  | `info`     | `debug`, `info`, `warn`, `error`                                     |

Each non-`stdout` sink has its own set of keys (`httpSinkUrl`, `s3SinkBucket`,
`influxdbHost`, ...) - see [`sinks/interfaces.go`][sinks-interfaces] for the
full list per sink.

## Event APIs

Kubernetes serves events under two API groups. The original `core/v1` Event is
still served and is not deprecated; `events.k8s.io/v1` was added later with a
different schema (`regarding` for `involvedObject`, `note` for `message`,
`reportingController` for `source`) and is what migrated reporters such as the
scheduler write to.

Eventrouter watches `core/v1`, which sees **every** event in the cluster: the
two groups are two views of the same stored objects and the API server converts
between them. The conversion is not lossless, though - an event written through
`events.k8s.io/v1` arrives over `core/v1` with an empty `source` and with no
`firstTimestamp` or `lastTimestamp`. Eventrouter therefore falls back to
`reportingComponent` for the component and to `eventTime` /
`series.lastObservedTime` for the time, so events from either API carry a
reporter and a real timestamp. Only the node stays unknown for
`events.k8s.io/v1` events, which name no host at all.

The Prometheus counters (`eventrouter_normal_total`,
`eventrouter_warnings_total`, `eventrouter_info_total`,
`eventrouter_unknown_total`) are labelled with `source` - the reporting node,
empty when the event names none - and `component`, the reporting controller.

## Logging

Events are written to **stdout** by the stdout sink (one JSON object per line).
The application's own logs are structured ([log/slog][slog]) and go to
**stderr**, so the two streams never get mixed. `log-format`/`log-level`
control the latter - see the [Configuration](#configuration) table above.

```
$ kubectl set env deployment/eventrouter -n kube-system LOG_LEVEL=debug
```

> **Note:** the `glog` flags (`-v`, `-logtostderr`, `-log_dir`, ...) were removed
> along with the `github.com/golang/glog` dependency. Passing them now makes the
> binary exit with a flag parsing error - use `log-level`/`LOG_LEVEL` instead.

[kubernetes]: https://github.com/kubernetes/kubernetes/ "Kubernetes"
[slog]: https://pkg.go.dev/log/slog "log/slog"
[configmap-example]: tests/eventrouter/eventrouter-with-configmap.yaml "ConfigMap example"
[sinks-interfaces]: sinks/interfaces.go "sink configuration keys"
