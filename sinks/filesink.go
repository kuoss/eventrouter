package sinks

import (
	"encoding/json"
	"log/slog"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	v1 "k8s.io/api/core/v1"
)

// FileSink writes one JSON event per line to a local file, the way
// StdoutSink does to stdout - but through a rotating writer
// (gopkg.in/natefinch/lumberjack.v2), so the file doesn't grow without
// bound. Path needs a volume mounted under it to be worth using at all:
// either to survive past the container's own lifetime, or just so a
// different sidecar (a log shipper, say) can read the same file over a
// volume they share - which also replaces the tee-to-logrotate sidecar
// pattern in tests/eventrouter/eventrouter-with-sidecar.yaml, since this
// does its own rotation.
type FileSink struct {
	logger *lumberjack.Logger
}

// FileSinkConfig configures a FileSink. Every field but Path is optional -
// lumberjack's own zero-value defaults apply: MaxSize 0 means 100
// megabytes; MaxBackups and MaxAge 0 mean keep every rotated file forever;
// Compress false leaves them uncompressed.
type FileSinkConfig struct {
	Path       string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// NewFileSink constructs a new FileSink, returned as an EventSinkInterface.
func NewFileSink(cfg FileSinkConfig) EventSinkInterface {
	return &FileSink{
		logger: &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		},
	}
}

// UpdateEvents implements the EventSinkInterface
func (f *FileSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	eData := NewEventData(eNew, eOld)
	line, err := json.Marshal(eData)
	if err != nil {
		slog.Warn("failed to json serialize event", "err", err)
		return
	}
	line = append(line, '\n')
	if _, err := f.logger.Write(line); err != nil {
		slog.Warn("failed to write event to file", "path", f.logger.Filename, "err", err)
	}
}
