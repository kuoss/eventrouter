package sinks

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	v1 "k8s.io/api/core/v1"
)

const maxMessageSize = 1046528

// EventHubSink sends events to an Azure Event Hub.
type EventHubSink struct {
	hub     *eventhub.Hub
	eventCh chan EventData
	wg      sync.WaitGroup
}

// NewEventHubSink constructs a new EventHubSink given an event hub connection string
// and buffering options.
func NewEventHubSink(connString string, bufferSize int) (*EventHubSink, error) {
	hub, err := eventhub.NewHubFromConnectionString(connString)
	if err != nil {
		return nil, err
	}

	eventCh := make(chan EventData, bufferSize)
	return &EventHubSink{hub: hub, eventCh: eventCh}, nil
}

// UpdateEvents implements the EventSinkInterface. It writes the event data to the channel.
// If the channel is full, the data will be dropped to prevent blocking.
func (h *EventHubSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	select {
	case h.eventCh <- NewEventData(eNew, eOld):
	default:
		log.Printf("Event channel is full, discarding event: %v", eNew)
	}
}

// Run sits in a loop, waiting for data to come in through h.eventCh,
// and forwarding them to the event hub. It also handles stop signal.
func (h *EventHubSink) Run(stopCh <-chan struct{}) {
	h.wg.Add(1)
	defer h.wg.Done()

	for {
		select {
		case evt := <-h.eventCh:
			events := []EventData{evt}

			for {
				select {
				case evt := <-h.eventCh:
					events = append(events, evt)
				default:
					h.drainEvents(events)
					break
				}
			}
		case <-stopCh:
			return
		}
	}
}

// drainEvents sends event data to the Event Hub.
func (h *EventHubSink) drainEvents(events []EventData) {
	var messageSize int
	var evts []*eventhub.Event
	for _, evt := range events {
		eJSONBytes, err := json.Marshal(evt)
		if err != nil {
			log.Printf("Failed to marshal event data: %v", err)
			return
		}
		log.Printf("Event data: %s", eJSONBytes)
		messageSize += len(eJSONBytes)
		if messageSize > maxMessageSize {
			h.sendBatch(evts)
			evts = nil
			messageSize = 0
		}
		evts = append(evts, eventhub.NewEvent(eJSONBytes))
	}
	h.sendBatch(evts)
}

func (h *EventHubSink) sendBatch(evts []*eventhub.Event) {
	if err := h.hub.SendBatch(context.Background(), eventhub.NewEventBatchIterator(evts...)); err != nil {
		log.Printf("Failed to send batch of %d events: %v", len(evts), err)
	}
}
