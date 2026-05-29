package monitoring

import (
	"testing"

	"project_smt6/app/models"
	"project_smt6/config"

	"github.com/gosnmp/gosnmp"
)

func TestParsePingOutput(t *testing.T) {
	output := `PING 127.0.0.1 (127.0.0.1) 56(84) bytes of data.
64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.052 ms
64 bytes from 127.0.0.1: icmp_seq=2 ttl=64 time=0.045 ms
64 bytes from 127.0.0.1: icmp_seq=3 ttl=64 time=0.079 ms

--- 127.0.0.1 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2026ms
rtt min/avg/max/mdev = 0.045/0.058/0.079/0.014 ms`

	result := parsePingOutput(output)
	if !result.success {
		t.Fatal("expected ping to be successful")
	}
	if result.packetLoss != 0 {
		t.Fatalf("expected 0 packet loss, got %v", result.packetLoss)
	}
	if result.latencyMS != 0.058 {
		t.Fatalf("expected avg latency 0.058, got %v", result.latencyMS)
	}
}

func TestParsePingOutputWithLoss(t *testing.T) {
	output := `3 packets transmitted, 2 received, 33.333% packet loss, time 2002ms
rtt min/avg/max/mdev = 9.100/10.200/11.300/0.500 ms`

	result := parsePingOutput(output)
	if !result.success {
		t.Fatal("partial packet loss should still be a reachable target")
	}
	if result.packetLoss != 33.333 {
		t.Fatalf("expected parsed packet loss, got %v", result.packetLoss)
	}
	if result.latencyMS != 10.200 {
		t.Fatalf("expected avg latency 10.2, got %v", result.latencyMS)
	}
}

func TestHealthFromPingUsesThresholds(t *testing.T) {
	service := NewService(nil, config.MonitoringConfig{
		Enabled: true,
		Ping: config.PingConfig{
			Enabled:                  true,
			WarningLatencyMS:         100,
			WarningPacketLossPercent: 20,
		},
	}, nil)

	online := service.healthFromPing(pingResult{success: true, latencyMS: 50, packetLoss: 0})
	if *online != models.DeviceStatusOnline {
		t.Fatalf("expected online, got %s", *online)
	}

	warning := service.healthFromPing(pingResult{success: true, latencyMS: 150, packetLoss: 0})
	if *warning != models.DeviceStatusWarning {
		t.Fatalf("expected warning from latency, got %s", *warning)
	}

	offline := service.healthFromPing(pingResult{success: false, packetLoss: 100})
	if *offline != models.DeviceStatusOffline {
		t.Fatalf("expected offline, got %s", *offline)
	}
}

func TestNumericSNMPValue(t *testing.T) {
	value, ok := numericSNMPValue([]byte("42.5"))
	if !ok || value != 42.5 {
		t.Fatalf("expected numeric byte value, got %v ok=%v", value, ok)
	}

	value, ok = numericSNMPValue(uint64(90))
	if !ok || value != 90 {
		t.Fatalf("expected numeric uint64 value, got %v ok=%v", value, ok)
	}

	if _, ok := numericSNMPValue([]byte("not-a-number")); ok {
		t.Fatal("non-numeric SNMP value must be rejected")
	}
}

func TestSNMPVersion(t *testing.T) {
	if snmpVersion("1") != gosnmp.Version1 {
		t.Fatal("expected SNMP v1")
	}
	if snmpVersion("2c") != gosnmp.Version2c {
		t.Fatal("expected SNMP v2c")
	}
	if snmpVersion("unknown") != gosnmp.Version2c {
		t.Fatal("unknown version should default to v2c")
	}
}
