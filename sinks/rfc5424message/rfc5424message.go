package rfc5424message

import (
	"fmt"
	"time"
)

type Message struct {
	Timestamp time.Time
	Hostname  string
	AppName   string
	Message   []byte
}

func (m Message) Bytes() []byte {
	s := fmt.Sprintf("<24>1 %s %s %s - - - %s", m.Timestamp.Format(time.RFC3339), m.Hostname, m.AppName, m.Message)
	fmt.Println(s)
	return []byte(fmt.Sprintf("%d %s", len(s), s))
}

// func createRFC5424Message(e *EventData) (string, error) {
// 	jsonBytes, err := json.Marshal(e)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to marshal event to JSON: %v", err)
// 	}

// 	// Build the RFC5424 message components
// 	priority := "<24>" // 24 indicates user-level messages, severity info
// 	version := "1"     // version of the syslog protocol
// 	timestamp := e.Event.LastTimestamp.Time.Format(time.RFC3339Nano)
// 	hostname := e.Event.Source.Host
// 	appName := e.Event.Source.Component
// 	procID := "-"
// 	msgID := "-"
// 	structuredData := "-"
// 	message := string(jsonBytes)

// 	// Form the full RFC5424 message
// 	rfc5424Message := fmt.Sprintf("%s%s %s %s %s %s %s %s %s",
// 		priority, version, timestamp, hostname, appName, procID, msgID, structuredData, message)

// 	return rfc5424Message, nil
// }
