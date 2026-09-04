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

`kubectl create` only works for a first install - to change something later (bump the
image tag, tweak the ConfigMap) without deleting and recreating everything, apply
[`deploy/`][deploy-dir] with kustomize instead, which `kubectl apply -k` re-runs safely:
```
$ kubectl apply -k github.com/kuoss/eventrouter/deploy?ref=main
```
(or clone the repo and run it against `deploy/` locally, e.g. from a kustomize overlay
that patches the image tag or replica count.)

The image is published for `linux/amd64`, `linux/arm64` and `linux/arm/v7`.

### Inspecting the output 
```
$ kubectl logs -f deployment/eventrouter -n kube-system 
``` 

## Configuration

Every config key already has a default, so eventrouter runs with no config
file at all. To change one, mount a ConfigMap with a `config.yaml` at
`/etc/eventrouter/` - [`deploy/deploy.yaml`][deploy-manifest] already does
this, so `kubectl edit configmap eventrouter -n kube-system` (then restart the
deployment to pick it up) is enough for most changes - or, for local runs,
copy [`config.example.yaml`][config-example] to `config.yaml` next to the
binary and edit that; with no `--config` flag, both paths are tried in that
order. A malformed file (present but not valid YAML) still fails startup, and
so does a missing one *if* `--config` names it explicitly; left to the
default search, a missing file does not.

> **Upgrading from a `config.json` deployment:** versions before 0.6 read
> `config.json`; this one reads `config.yaml` and does not fall back to the
> old file. A ConfigMap still keyed `config.json` after upgrading produces no
> error - every key silently reverts to its default (a single `stdout` sink
> among them) - so a deployment relying on a non-default sink needs its
> ConfigMap key renamed to `config.yaml`, with the content converted to YAML
> (see [`config.example.yaml`][config-example]), as part of the upgrade.

| config key          | default             | description                                                                                 |
| -------------------- | ------------------- | -------------------------------------------------------------------------------------------|
| `kubeconfig`         | *(empty)*           | path to a kubeconfig file; empty uses the in-cluster service account (env: `KUBECONFIG`, only when this is empty) |
| `sinks`              | `[{type: stdout}]`  | a list, each entry a `type` plus its own settings                                          |
| `enable-prometheus`  | `true`              | exposes `/metrics` and the event counters                                                  |
| `log.format`         | `json`              | `json`, `text`                                                                              |
| `log.level`          | `info`              | `debug`, `info`, `warn`, `error`                                                            |

Each `sinks` entry's `type` (`stdout`, `http`, `s3sink`, `influxdb`,
`filesink`) picks what the rest of that entry configures - see
[`config.example.yaml`][config-example] for the full, commented settings per
type. Listing the same type more than once configures more than one
instance of it, each with independent settings - two different HTTP
endpoints, say, which one shared block could never express:
```yaml
sinks:
  - type: http
    url: http://a
  - type: http
    url: http://b
```

An explicit `sinks: []` means zero sinks - eventrouter still runs and counts
events for `/metrics`, but forwards none of them anywhere, and says so in
its own logs on startup rather than doing it silently.

If `sinks` is omitted or set to `null`, it defaults to one `stdout` sink. The
value must otherwise be a list of sink objects: scalar values such as
`sinks: stdout`, lists of names such as `sinks: [stdout]`, or non-object list
entries are invalid configuration and stop startup with an error.

## Event APIs

Kubernetes serves events under two API groups. The original `core/v1` Event is
still served and is not deprecated; `events.k8s.io/v1` was added later with a
different schema (`regarding` for `involvedObject`, `note` for `message`,
`reportingController` for `source`) and is what migrated reporters such as the
scheduler write to.

Eventrouter watches `core/v1`, which sees **every** event in the cluster: the
two groups are two views of the same stored objects and `core/v1.Event`
already has room for both APIs' fields (see [`docs/event.md`][event-doc] for
the field-by-field mapping and why watching one side is enough). A reporter
only ever populates the fields its own API knows about, though - an event
written through `events.k8s.io/v1` arrives with an empty `source` and no
`firstTimestamp`/`lastTimestamp`, since that reporter never sets them.
Eventrouter therefore falls back to `reportingComponent` for the component
and to `eventTime`/`series.lastObservedTime` for the time, so events from
either API carry a reporter and a real timestamp. Only the node stays
unknown for `events.k8s.io/v1` events, which name no host at all.

The Prometheus counters (`eventrouter_normal_total`,
`eventrouter_warnings_total`, `eventrouter_info_total`,
`eventrouter_unknown_total`) are labelled with `source` - the reporting node,
empty when the event names none - and `component`, the reporting controller.

## Logging

### Event output

Events are written to **stdout** by the stdout sink (one JSON object per
line). Every sink writes each event as `{"verb": "ADDED"|"UPDATED", "event":
<the Kubernetes Event>}` - `ADDED` the first time eventrouter sees it,
`UPDATED` on a repeat (kubelet bumps `count`/`lastTimestamp` on the same
object; an events.k8s.io/v1 reporter attaches a `series` instead - see
[Event APIs](#event-apis)). There is no `old_event`/before-snapshot: a
repeat's own fields already say what changed. See
[`tests/sample/pod-log.ndjson`][sample-log] for a real ADDED/UPDATED pair from
each API shape.

### Application logs

The application's own logs are structured and go to **stderr**, so they
never get mixed with event output on stdout. `log.format`/`log.level`
control them - see the [Configuration](#configuration) table above.

```
$ kubectl edit configmap eventrouter -n kube-system   # set log.level: debug
$ kubectl rollout restart deployment/eventrouter -n kube-system
```

> **Note:** the `glog` flags (`-v`, `-logtostderr`, `-log_dir`, ...) were removed
> along with the `github.com/golang/glog` dependency. Passing them now makes the
> binary exit with a flag parsing error - use `log.level` instead.

[kubernetes]: https://github.com/kubernetes/kubernetes/ "Kubernetes"
[deploy-manifest]: deploy/deploy.yaml "deployment manifest"
[deploy-dir]: deploy/ "deploy/"
[event-doc]: docs/event.md "core/v1 vs events.k8s.io/v1"
[config-example]: config.example.yaml "annotated config reference"
[sample-log]: tests/sample/pod-log.ndjson "sample pod log"
