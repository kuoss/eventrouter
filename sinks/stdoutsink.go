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
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
)

// StdoutSink is the other basic sink
// By writing raw JSON events to stdout, we will get automated indexing which
// can be queried in Kibana. Application logs are kept off stdout (they go to
// stderr) so that this stream contains events only.
type StdoutSink struct{}

// NewStdoutSink will create a new StdoutSink with default options, returned as
// an EventSinkInterface
func NewStdoutSink() EventSinkInterface {
	return &StdoutSink{}
}

// UpdateEvents implements the EventSinkInterface
func (gs *StdoutSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	eData := NewEventData(eNew, eOld)
	if eJSONBytes, err := json.Marshal(eData); err == nil {
		fmt.Println(string(eJSONBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Failed to json serialize event: %v", err)
	}
}
