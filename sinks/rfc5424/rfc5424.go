package rfc5424

import (
	"fmt"
	"strings"
	"time"
)

// nilValue is the RFC5424 NILVALUE, which stands in for a header field the
// sender cannot supply. An event written through the events.k8s.io/v1 API
// names no host, so an absent hostname is routine rather than exceptional.
const nilValue = "-"

type Message struct {
	Timestamp time.Time
	Hostname  string
	AppName   string
	Message   string
}

func (m *Message) Bytes() []byte {
	s := fmt.Sprintf("<24>1 %s %s %s - - - %s", m.Timestamp.Format(time.RFC3339Nano),
		orNilValue(m.Hostname), orNilValue(m.AppName), m.Message)
	return []byte(fmt.Sprintf("%d %s", len(s), s))
}

func NewFromBytes(data []byte) (*Message, error) {
	parts := strings.SplitN(string(data), " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: missing length prefix")
	}

	syslogParts := strings.SplitN(parts[1], " ", 7)
	if len(syslogParts) < 7 {
		return nil, fmt.Errorf("invalid syslog format")
	}

	timestamp, err := time.Parse(time.RFC3339Nano, syslogParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %w", err)
	}

	return &Message{
		Timestamp: timestamp,
		Hostname:  fromNilValue(syslogParts[2]),
		AppName:   fromNilValue(syslogParts[3]),
		Message:   strings.TrimPrefix(syslogParts[6], "- "),
	}, nil
}

// orNilValue keeps the header well-formed when a field is unknown: leaving it
// empty would collapse two spaces into one and shift every field after it.
func orNilValue(s string) string {
	if s == "" {
		return nilValue
	}
	return s
}

func fromNilValue(s string) string {
	if s == nilValue {
		return ""
	}
	return s
}
