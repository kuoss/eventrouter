# Testing

## Unit tests

```
$ make test        # go test with coverage
$ make test-race    # same, under the race detector (needs a C toolchain)
$ make cover         # make test, then coverage per function
```

Tests are table-driven throughout. The one shared fixture worth knowing about
is [`internal/kubeevent/kubeeventtest`][kubeeventtest]: an Option-based
builder for a `core/v1` event shaped the way a `core/v1` reporter (kubelet)
writes it, or the way the API server returns one that an
`events.k8s.io/v1` reporter (the scheduler, for one) wrote - see
[`docs/event.md`](event.md) for why those two shapes differ.
`internal/kubeevent`, `internal/router` and `sinks` each exercise both
shapes through it rather than hand-rolling their own; add a field there
(not a one-off literal) if a test needs to vary something it does not
expose yet.

`make checks` (lint, workflow lint, tests, image build, vulnerability scan)
is what pull requests run - see [`CONTRIBUTING.md`](../CONTRIBUTING.md) for
the rest of the development loop.

## Sample output

```
$ make sample
```

regenerates [`tests/sample/pod-log.ndjson`][sample-log] via
[`tests/sample/gen`][sample-gen]: a deterministic, checked-in example of what
`kubectl logs` shows for the default config - eventrouter's own startup logs,
plus one `core/v1` and one `events.k8s.io/v1` event each shown as both
`ADDED` and `UPDATED`. It is real output, produced by the same code the
service ships (`sinks.NewEventData`, the same JSON handler `internal/logging`
installs), not a hand-written mockup - regenerate it whenever a log message's
wording, or the `EventData` JSON shape, changes.

## Manual / kind-cluster tests

These need a running cluster ([kind][kind] by default) and are not run in CI;
`make kind-create`/`make kind-delete` (repo root `Makefile`) manage the
cluster itself.

- `make kind-deploy` builds nothing on its own - it loads an already-built
  image into kind and applies [`tests/eventrouter/eventrouter-with-sidecar.yaml`][sidecar-manifest],
  then tails the pod's logs. Use it to confirm a change works against a real
  API server, not just the fake clientset unit tests use.
- [`tests/eventrouter/`][eventrouter-manifests] also has a ConfigMap-mounting
  variant of the deployment - see [`deploy/deploy.yaml`][deploy-manifest] at
  the repo root, which now ships that pattern by default.
- [`tests/eventxxx/`][eventxxx] runs no eventrouter code at all - it is a
  small sandbox (own `Makefile`) for deploying a different tool
  (Fluent Bit's `kubernetes_events` input, or [kubernetes-event-exporter][event-exporter])
  on the same kind cluster to eyeball its output next to eventrouter's.

[kubeeventtest]: ../internal/kubeevent/kubeeventtest/kubeeventtest.go "shared event fixture builder"
[sample-log]: ../tests/sample/pod-log.ndjson "sample pod log"
[sample-gen]: ../tests/sample/gen/main.go "sample generator"
[kind]: https://kind.sigs.k8s.io/ "kind"
[sidecar-manifest]: ../tests/eventrouter/eventrouter-with-sidecar.yaml "kind smoke-test manifest"
[eventrouter-manifests]: ../tests/eventrouter/ "tests/eventrouter"
[deploy-manifest]: ../deploy/deploy.yaml "deployment manifest"
[eventxxx]: ../tests/eventxxx/ "tests/eventxxx"
[event-exporter]: https://github.com/opsgenie/kubernetes-event-exporter "kubernetes-event-exporter"
