package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type cleanHandler struct {
	out    io.Writer
	level  slog.Leveler
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func newCleanHandler(out io.Writer, level slog.Leveler) slog.Handler {
	return &cleanHandler{
		out:   out,
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *cleanHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.level != nil {
		min = h.level.Level()
	}
	return level >= min
}

func (h *cleanHandler) Handle(_ context.Context, record slog.Record) error {
	icon, msg := cleanMessage(record.Level, record.Message)
	if record.Time.IsZero() {
		record.Time = time.Now()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s", record.Time.Format("15:04:05"), icon, msg)

	writeAttr := func(attr slog.Attr) bool {
		attr.Value = attr.Value.Resolve()
		if attr.Equal(slog.Attr{}) {
			return true
		}
		key := h.attrKey(attr.Key)
		if isSensitiveKey(key) {
			b.WriteString(" ")
			b.WriteString(key)
			b.WriteString("=<redacted>")
			return true
		}
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(formatValue(attr.Value))
		return true
	}

	for _, attr := range h.attrs {
		writeAttr(attr)
	}
	record.Attrs(writeAttr)
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *cleanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *cleanHandler) WithGroup(name string) slog.Handler {
	if strings.TrimSpace(name) == "" {
		return h
	}
	next := *h
	next.groups = append(append([]string{}, h.groups...), name)
	return &next
}

func (h *cleanHandler) attrKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(append(append([]string{}, h.groups...), key), ".")
}

func cleanMessage(level slog.Level, msg string) (string, string) {
	switch {
	case strings.HasPrefix(msg, "[OK] "):
		return "✅", strings.TrimPrefix(msg, "[OK] ")
	case strings.HasPrefix(msg, "[WARN] "):
		return "⚠️", strings.TrimPrefix(msg, "[WARN] ")
	case strings.HasPrefix(msg, "[FAIL] "):
		return "❌", strings.TrimPrefix(msg, "[FAIL] ")
	}

	switch {
	case level >= slog.LevelError:
		return "❌", msg
	case level >= slog.LevelWarn:
		return "⚠️", msg
	case level <= slog.LevelDebug:
		return "🔎", msg
	default:
		return "ℹ️", msg
	}
}

func formatValue(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return quoteIfNeeded(value.String())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339)
	case slog.KindBool:
		return fmt.Sprintf("%t", value.Bool())
	case slog.KindInt64:
		return fmt.Sprintf("%d", value.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", value.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%g", value.Float64())
	default:
		return quoteIfNeeded(fmt.Sprint(value.Any()))
	}
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\r=:") {
		return fmt.Sprintf("%q", value)
	}
	return value
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token")
}
