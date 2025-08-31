package logging

import (
	"log/slog"
	"os"
	"strings"
)

var Logger *slog.Logger

func New(logLeveL string, json bool) *slog.Logger {
	level := parseLevel(logLeveL)
	opts := &slog.HandlerOptions{Level: level, AddSource: true}
	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	s = strings.ToLower(s)
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
