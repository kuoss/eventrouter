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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	v1 "k8s.io/api/core/v1"
)

// EventData encodes an eventrouter event and previous event, with a verb for
// whether the event is created or updated.
type EventData struct {
	Verb     string    `json:"verb"`
	Event    *v1.Event `json:"event"`
	OldEvent *v1.Event `json:"old_event,omitempty"`
}

// NewEventData constructs an EventData struct from an old and new event,
// setting the verb accordingly
func NewEventData(eNew *v1.Event, eOld *v1.Event) EventData {
	var eData EventData
	if eOld == nil {
		eData = EventData{
			Verb:  "ADDED",
			Event: eNew,
		}
	} else {
		eData = EventData{
			Verb:     "UPDATED",
			Event:    eNew,
			OldEvent: eOld,
		}
	}

	return eData
}

// WriteRFC5424 writes the current event data to the given io.Writer using
// RFC5424 (syslog over TCP) syntax.
func (e *EventData) WriteRFC5424(w io.Writer) (int64, error) {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal event to JSON: %v", err)
	}

	// Build the RFC5424 message
	priority := "<14>" // 14 indicates user-level messages, severity info
	version := "1"     // version of the syslog protocol
	timestamp := e.Event.LastTimestamp.Time.Format(time.RFC3339)
	hostname := e.Event.Source.Host
	appName := e.Event.Source.Component
	procID := "-"
	msgID := "-"
	structuredData := "-"
	message := string(jsonBytes)

	rfc5424Message := fmt.Sprintf("%s%s %s %s %s %s %s %s %s",
		priority, version, timestamp, hostname, appName, procID, msgID, structuredData, message)

	written, err := w.Write([]byte(rfc5424Message))
	return int64(written), err
}

// WriteFlattenedJSON writes the json to the file in the below format
// 1) Flattens the json into a not nested key:value
// 2) Convert the json into snake format
// Eg: {"event_involved_object_kind":"pod", "event_metadata_namespace":"kube-system"}
func (e *EventData) WriteFlattenedJSON(w io.Writer) (int64, error) {
	eventJSON, err := json.Marshal(e)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal event to JSON: %v", err)
	}

	result, err := explodeJSONStr(string(eventJSON), "_")
	if err != nil {
		return 0, fmt.Errorf("failed to flatten JSON: %v", err)
	}

	written, err := w.Write([]byte(result))
	return int64(written), err
}

func explodeJSONStr(jsonStr, separator string) (string, error) {
	var inputMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &inputMap)
	if err != nil {
		return "", errors.New("failed to unmarshal JSON")
	}

	flatMap := make(map[string]interface{})
	flatten("", inputMap, flatMap, separator)

	flatJSON, err := json.Marshal(flatMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal flattened JSON: %v", err)
	}

	return string(flatJSON), nil
}

// flatten is a helper function that recursively flattens JSON.
func flatten(prefix string, input interface{}, flatMap map[string]interface{}, separator string) {
	if nestedMap, ok := input.(map[string]interface{}); ok {
		for k, v := range nestedMap {
			newKey := k
			if prefix != "" {
				newKey = prefix + separator + k
			}
			flatten(newKey, v, flatMap, separator)
		}
	} else {
		flatMap[prefix] = input
	}
}
