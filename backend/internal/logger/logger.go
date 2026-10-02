package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	h := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: l})
	return slog.New(h)
}
