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

## Logging

Events are written to **stdout** by the stdout sink (one JSON object per line).
The application's own logs are structured ([log/slog][slog]) and go to
**stderr**, so the two streams never get mixed.

| config key   | env var      | default | values                          |
| ------------ | ------------ | ------- | ------------------------------- |
| `log-format` | `LOG_FORMAT` | `json`  | `json`, `text`                  |
| `log-level`  | `LOG_LEVEL`  | `info`  | `debug`, `info`, `warn`, `error`|

```
$ kubectl set env deployment/eventrouter -n kube-system LOG_LEVEL=debug
```

> **Note:** the `glog` flags (`-v`, `-logtostderr`, `-log_dir`, ...) were removed
> along with the `github.com/golang/glog` dependency. Passing them now makes the
> binary exit with a flag parsing error - use `log-level`/`LOG_LEVEL` instead.

[kubernetes]: https://github.com/kubernetes/kubernetes/ "Kubernetes"
[slog]: https://pkg.go.dev/log/slog "log/slog"
