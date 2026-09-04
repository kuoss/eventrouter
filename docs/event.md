# core/v1 vs events.k8s.io/v1

Kubernetes serves events under two API groups, and this document is the deep
dive behind the summary in [`README.md`'s Event APIs section][readme-events]:
why eventrouter watches `core/v1`, and why that is enough to see events from
either API without watching both.

## Two names for the same stored object

`core/v1` Event is the original API and is not deprecated. `events.k8s.io/v1`
was added later (GA since Kubernetes 1.19) with more structured field names
(`regarding` instead of `involvedObject`, `note` instead of `message`,
`reportingController` instead of `source`) and is what newer reporters, such
as the scheduler, write through.

They are not two separate stores of events - the API server converts between
them on the fly, so a watch on either API sees every event in the cluster
regardless of which API its reporter used to write it.

## The field mapping is total, not lossy

It would be easy to assume the newer API's schema is strictly richer and that
watching it would be the more complete choice. It is not: `core/v1.Event`
already carries every field either schema needs, confirmed directly from the
types this module vends (`k8s.io/api/core/v1`, module version pinned in
`go.mod`):

```go
type Event struct {
    // present only under these names on core/v1
    InvolvedObject ObjectReference
    Message        string
    Source         EventSource
    FirstTimestamp metav1.Time
    LastTimestamp  metav1.Time
    Count          int32

    // present under the *same* names on events.k8s.io/v1
    EventTime            metav1.MicroTime
    Series               *EventSeries
    Action               string
    ReportingController  string
    ReportingInstance    string
}
```

`events.k8s.io/v1.Event` mirrors this exactly from the other side: `EventTime`,
`Series`, `Action`, `ReportingController` and `ReportingInstance` keep their
plain names, while the `core/v1`-only fields are preserved under
`DeprecatedSource`, `DeprecatedFirstTimestamp`, `DeprecatedLastTimestamp` and
`DeprecatedCount` - fields whose own doc comment in `k8s.io/api/events/v1`
says they exist "assuring backward compatibility with core.v1 Event type."

Kubernetes' own generated conversion
([`pkg/apis/events/v1/zz_generated.conversion.go`][zz-generated],
[`pkg/apis/events/v1/conversion.go`][conversion]) confirms this is exactly
how the two directions work: `EventTime`/`Series`/`Action`/
`ReportingController`/`ReportingInstance` are copied straight across because
both types carry them under identical names, while `Regarding`/`Note`/
`DeprecatedSource`/`DeprecatedFirstTimestamp`/`DeprecatedLastTimestamp`/
`DeprecatedCount` need (and get) a manual rename in the generated converter.

Nothing is dropped either direction. The two APIs are the same information
under two field-naming conventions, not two schemas of differing richness.

## What this means for eventrouter

Because the mapping is total, **watching either API alone already sees
everything** - there is no reason for eventrouter to watch both, and no
migration to `events.k8s.io/v1` would gain any field it cannot already read
today. `internal/kubeevent`'s `Component`/`Host`/`Timestamp` helpers read
whichever of the two names is populated on the `core/v1` object eventrouter
already watches:

- `Component`: `Source.Component`, falling back to `ReportingController`.
- `Timestamp`: `LastTimestamp`, then `Series.LastObservedTime`, then
  `EventTime`, then `FirstTimestamp`, then the object's `creationTimestamp`.
- `Host`: `Source.Host` only - `ReportingInstance` is a controller instance
  ID, not a hostname, so it is deliberately not used as a fallback.

Switching the watched API from `core/v1` to `events.k8s.io/v1` would just
rename these same fields (and, as a real consequence, change the JSON shape
every sink writes) for no functional gain, so eventrouter stays on `core/v1`.

[readme-events]: ../README.md#event-apis "Event APIs"
[zz-generated]: https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/events/v1/zz_generated.conversion.go "generated Event conversion"
[conversion]: https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/events/v1/conversion.go "manual Event conversion overrides"
