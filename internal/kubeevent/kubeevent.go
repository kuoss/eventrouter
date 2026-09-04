/*
Copyright 2025 Kuoss

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

// Package kubeevent reads the fields of a core/v1 Event in a way that also
// works for events written through the events.k8s.io/v1 API.
//
// The two API groups are two views of the same stored objects - the API server
// converts between them - so a core/v1 watch still sees every event in the
// cluster no matter which API its reporter used. The conversion is not
// lossless, though. An event written through events.k8s.io/v1 arrives over
// core/v1 with:
//
//   - an empty source, because that API has no required equivalent. The
//     reporter is in reportingComponent/reportingInstance instead.
//   - no firstTimestamp and no lastTimestamp, because that API carries
//     eventTime and series.lastObservedTime instead.
//
// Reading those fields directly therefore yields an empty component and a zero
// timestamp for every event from a reporter that has migrated (the scheduler,
// for one). Component and Timestamp read whichever field is populated, so
// callers do not have to care which API the reporter used; Host does not fall
// back, for the reason given on its own doc.
package kubeevent

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Component returns the name of the controller that reported the event, or ""
// if the event names neither a source component nor a reporting controller.
func Component(e *corev1.Event) string {
	if e.Source.Component != "" {
		return e.Source.Component
	}
	return e.ReportingController
}

// Host returns the node the event was reported from, or "" when the event does
// not say.
//
// There is deliberately no fallback to reportingInstance: that field holds an
// instance ID rather than a hostname - the scheduler reports its own pod name
// there - so treating it as a host would be wrong more often than right. An
// empty result is the honest answer for a reporter that supplies no host.
func Host(e *corev1.Event) string {
	return e.Source.Host
}

// Timestamp returns the most recent time the event was observed, and the zero
// time if the event carries no time at all.
//
// The fields are tried newest-meaning first: lastTimestamp is what core/v1
// reporters maintain, series.lastObservedTime is what an events.k8s.io/v1
// reporter updates on every repeat, and eventTime is the first observation of
// a singleton event. firstTimestamp and the object's creationTimestamp are
// last resorts, so that an event missing every event-level time still sorts
// somewhere sane rather than at the start of the epoch.
func Timestamp(e *corev1.Event) time.Time {
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time
	}
	if e.Series != nil && !e.Series.LastObservedTime.IsZero() {
		return e.Series.LastObservedTime.Time
	}
	if !e.EventTime.IsZero() {
		return e.EventTime.Time
	}
	if !e.FirstTimestamp.IsZero() {
		return e.FirstTimestamp.Time
	}
	return e.CreationTimestamp.Time
}
