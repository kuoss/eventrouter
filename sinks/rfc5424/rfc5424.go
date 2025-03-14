package rfc5424

import (
	"fmt"
	"strings"
	"time"
)

type Message struct {
	Timestamp time.Time
	Hostname  string
	AppName   string
	Message   string
}

func (m *Message) Bytes() []byte {
	s := fmt.Sprintf("<24>1 %s %s %s - - - %s", m.Timestamp.Format(time.RFC3339Nano), m.Hostname, m.AppName, m.Message)
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

	// 타임스탬프 파싱 (RFC3339Nano)
	timestamp, err := time.Parse(time.RFC3339Nano, syslogParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %v", err)
	}

	// 시스템 로컬 타임존으로 변환
	timestamp = timestamp.In(time.Local)

	return &Message{
		Timestamp: timestamp,
		Hostname:  syslogParts[2],
		AppName:   syslogParts[3],
		Message:   strings.TrimPrefix(syslogParts[6], "- "),
	}, nil
}
