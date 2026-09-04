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

	"github.com/kuoss/eventrouter/internal/kubeevent"
	"github.com/kuoss/eventrouter/sinks/rfc5424"
	corev1 "k8s.io/api/core/v1"
)

// EventData encodes an eventrouter event with a verb for whether it was just
// created or is a repeat of one already seen.
type EventData struct {
	Verb  string        `json:"verb"`
	Event *corev1.Event `json:"event"`
}

// NewEventData constructs an EventData struct for eNew, setting the verb from
// whether eOld - the previously seen version of the same event, or nil for
// one seen for the first time - is present. eOld only ever decides the verb:
// it is not itself written anywhere. A repeat's own event fields (count,
// lastTimestamp or series) already say what changed, so carrying the whole
// previous snapshot alongside it would roughly double every repeat's payload
// to say very little more.
func NewEventData(eNew *corev1.Event, eOld *corev1.Event) EventData {
	verb := "ADDED"
	if eOld != nil {
		verb = "UPDATED"
	}
	return EventData{Verb: verb, Event: eNew}
}

// WriteRFC5424 writes the current event data to the given io.Writer using
// RFC5424 (syslog over TCP) syntax.
func (e *EventData) WriteRFC5424(w io.Writer) (int64, error) {
	var eJSONBytes []byte
	var err error
	if eJSONBytes, err = json.Marshal(e); err != nil {
		return 0, fmt.Errorf("failed to json serialize event: %w", err)
	}
	// Each message should look like an RFC5424 syslog message:
	// <NumberOfBytes/ASCII encoded integer><Space character><RFC5424 message:NumberOfBytes long>
	//
	// Note: There are some restrictions on length and character space for
	// Hostname and AppName, see
	// https://github.com/crewjam/rfc5424/blob/master/marshal.go#L90. There's no
	// attempt at trying to clean them up here because hostnames and component
	// names already adhere to this convention in practice.
	msg := rfc5424.Message{
		Timestamp: kubeevent.Timestamp(e.Event),
		Hostname:  kubeevent.Host(e.Event),
		AppName:   kubeevent.Component(e.Event),
		Message:   string(eJSONBytes),
	}

	written, err := w.Write(msg.Bytes())
	return int64(written), err
}

// WriteFlattenedJSON writes the json to the file in the below format
// 1) Flattens the json into a not nested key:value
// 2) Convert the json into snake format
// Eg: {"event_involved_object_kind":"pod", "event_metadata_namespace":"kube-system"}
func (e *EventData) WriteFlattenedJSON(w io.Writer) (int64, error) {
	eJSONBytes, err := json.Marshal(e)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	result, err := explodeJSONStr(string(eJSONBytes), "_")
	if err != nil {
		return 0, fmt.Errorf("failed to flatten JSON: %w", err)
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
		return "", fmt.Errorf("failed to marshal flattened JSON: %w", err)
	}

	return string(flatJSON), nil
}

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
