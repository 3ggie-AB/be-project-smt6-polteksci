package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type cleanHandler struct {
	out   io.Writer
	level slog.Level
	mu    *sync.Mutex
}

func NewLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "debug" {
		level = slog.LevelDebug
	}
	return slog.New(&cleanHandler{out: os.Stdout, level: level, mu: &sync.Mutex{}})
}

func (h *cleanHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *cleanHandler) Handle(_ context.Context, record slog.Record) error {
	icon, message := cleanMessage(record.Level, record.Message)

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s", record.Time.Format("15:04:05"), icon, message)
	record.Attrs(func(attr slog.Attr) bool {
		b.WriteString(" ")
		b.WriteString(attr.Key)
		b.WriteString("=")
		b.WriteString(fmt.Sprintf("%q", fmt.Sprint(attr.Value.Any())))
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func cleanMessage(level slog.Level, message string) (string, string) {
	for _, prefix := range []string{"✅ ", "⚠️ ", "❌ "} {
		if strings.HasPrefix(message, prefix) {
			return strings.TrimSpace(prefix), strings.TrimPrefix(message, prefix)
		}
	}
	if level >= slog.LevelError {
		return "❌", message
	}
	if level >= slog.LevelWarn {
		return "⚠️", message
	}
	return "ℹ️", message
}

func (h *cleanHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *cleanHandler) WithGroup(_ string) slog.Handler {
	return h
}
