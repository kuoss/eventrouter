/*
Copyright 2017 The Contributors

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

// Package logging installs the structured logger used by the whole program.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// logLevel is the level of the logger installed by Setup. It is a LevelVar so
// that the level stays adjustable after the logger is installed.
var logLevel = new(slog.LevelVar)

// Setup installs the structured logger used by the whole program.
//
// Logs always go to stderr: stdout belongs to the stdout sink, which prints one
// JSON event per line. JSON is the default format because log collectors
// (Fluentd/Elasticsearch and friends) index it without a custom parser; "text"
// is there for reading logs by eye during development.
//
// Until this runs, the slog default logger already writes to stderr at info
// level, so failures that happen while the config is being loaded are visible.
func Setup(format, level string) {
	logLevel.Set(parseLevel(level))

	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// parseLevel maps a config value ("debug", "info", "warn", "error") to a slog
// level, falling back to info for anything else.
func parseLevel(level string) slog.Level {
	if level == "" {
		return slog.LevelInfo
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		slog.Warn("unknown log level, using info", "logLevel", level)
		return slog.LevelInfo
	}
	return l
}
