eventrouter integration tests
=============================

The integration tests should be run manually.

`sample/pod-log.ndjson` is a checked-in example of what `kubectl logs` shows
for the default config: eventrouter's own startup logs interleaved with one
Kubernetes Event as the stdout sink prints it. Regenerate it with
`make sample` after changing a log message or the EventData JSON shape.





