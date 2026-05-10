package collector

import (
	"context"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
)

var syslogPRI = regexp.MustCompile(`^<(\d+)>`)

type SyslogReceiver struct {
	cfg    config.SyslogConfig
	sink   MetricSink
	events EventPublisher
	logger *slog.Logger
}

func NewSyslogReceiver(cfg config.SyslogConfig, sink MetricSink, events EventPublisher, logger *slog.Logger) *SyslogReceiver {
	return &SyslogReceiver{cfg: cfg, sink: sink, events: events, logger: logger}
}

func (r *SyslogReceiver) Run(ctx context.Context) {
	if !r.cfg.Enabled {
		r.logger.Warn("[WARN] Syslog receiver disabled")
		return
	}
	if r.cfg.Address == "" {
		r.cfg.Address = ":5514"
	}

	addr, err := net.ResolveUDPAddr("udp", r.cfg.Address)
	if err != nil {
		r.logger.Error("Syslog address resolve failed", "detail", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		r.logger.Error("Syslog receiver start failed", "address", r.cfg.Address, "detail", err)
		return
	}
	defer conn.Close()

	r.logger.Info("[OK] Syslog receiver started", "address", r.cfg.Address)
	buf := make([]byte, 64*1024)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, remote, err := conn.ReadFromUDP(buf)
		select {
		case <-ctx.Done():
			r.logger.Info("[OK] Syslog receiver stopped")
			return
		default:
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			r.logger.Error("Syslog packet read failed", "detail", err)
			continue
		}

		message := strings.TrimSpace(string(buf[:n]))
		event := parseSyslog(remote.IP.String(), message)
		r.sink.WriteSyslog(ctx, event)
		r.publishImportantEvent(event)
	}
}

func parseSyslog(ip, message string) domain.SyslogEvent {
	facility := "unknown"
	severity := "info"
	if match := syslogPRI.FindStringSubmatch(message); len(match) == 2 {
		priority := atoi(match[1])
		facility = syslogFacility(priority / 8)
		severity = syslogSeverity(priority % 8)
		message = strings.TrimSpace(syslogPRI.ReplaceAllString(message, ""))
	}
	return domain.SyslogEvent{
		Workspace: "default",
		IP:        ip,
		Facility:  facility,
		Severity:  severity,
		Hostname:  ip,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func (r *SyslogReceiver) publishImportantEvent(event domain.SyslogEvent) {
	lower := strings.ToLower(event.Message)
	if !(strings.Contains(lower, "down") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "critical") ||
		strings.Contains(lower, "error")) {
		return
	}
	r.events.Publish(domain.RealtimeEvent{
		Type:      "syslog.alert",
		Severity:  event.Severity,
		Workspace: event.Workspace,
		IP:        event.IP,
		Title:     "Syslog alert",
		Message:   event.Message,
		Attributes: map[string]any{
			"facility": event.Facility,
			"hostname": event.Hostname,
		},
	})
}

func syslogFacility(code int) string {
	names := []string{
		"kern", "user", "mail", "daemon", "auth", "syslog", "lpr", "news",
		"uucp", "cron", "authpriv", "ftp", "ntp", "security", "console", "solaris-cron",
		"local0", "local1", "local2", "local3", "local4", "local5", "local6", "local7",
	}
	if code >= 0 && code < len(names) {
		return names[code]
	}
	return "unknown"
}

func syslogSeverity(code int) string {
	names := []string{"emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"}
	if code >= 0 && code < len(names) {
		return names[code]
	}
	return "info"
}

func atoi(value string) int {
	var out int
	for _, r := range value {
		if r < '0' || r > '9' {
			return out
		}
		out = out*10 + int(r-'0')
	}
	return out
}
