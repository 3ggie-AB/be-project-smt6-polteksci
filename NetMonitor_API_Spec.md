# NetMonitor API Specification

**Version:** v2.0  
**Base URL:** `http://localhost:8080`  
**Stack:** Go + Gin + MySQL + InfluxDB  
**Auth:** Local email/password login + JWT Bearer token  

## Public Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness probe |
| `POST` | `/api/auth/login` | Login with email and password, returns JWT |
| `POST` | `/api/login` | Compatibility alias for local login |

First bootstrap login:

```json
{
  "email": "admin@netmonitor.local",
  "password": "admin123"
}
```

If no user exists, credentials must match `ADMIN_EMAIL` and `ADMIN_PASSWORD`; that first user becomes `SUPER_ADMIN`.

## Authenticated Endpoints

Send `Authorization: Bearer <jwt>`.

| Method | Path | Roles | Description |
| --- | --- | --- | --- |
| `GET` | `/api/me` | all | Current JWT user context |
| `GET` | `/api/stream` | all | SSE realtime monitoring stream |
| `GET` | `/api/devices` | all | List workspace devices |
| `POST` | `/api/devices` | `SUPER_ADMIN`, `ADMIN` | Create monitored device |
| `DELETE` | `/api/devices/:id` | `SUPER_ADMIN`, `ADMIN` | Delete device |
| `GET` | `/api/notifications` | all | List unread notifications |
| `POST` | `/api/notifications/:id/read` | all | Mark notification as read |
| `GET` | `/api/ml/features/:device_id` | all | Return ONNX-ready feature vector |

SSE can also accept the token as a query parameter for browser `EventSource`:

```text
GET /api/stream?access_token=<jwt>
```

## Create Device

```json
{
  "name": "AP Lobby",
  "ip_address": "192.168.10.20",
  "vendor": "ruijie",
  "device_type": "ap",
  "tcp_port": 443,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "snmp_community": "public"
}
```

## Realtime Events

```json
{
  "type": "latency.high",
  "severity": "warning",
  "workspace": "default",
  "device_id": 1,
  "ip": "192.168.10.20",
  "title": "High latency",
  "message": "AP Lobby latency is 184.25 ms",
  "attributes": {
    "latency_ms": 184.25
  },
  "occurred_at": "2026-05-10T10:00:00Z"
}
```

Event types:

- `ap.down`
- `latency.high`
- `packet_loss.high`
- `tcp.service_down`
- `anomaly.detected`
- `syslog.alert`

## InfluxDB Measurements

| Measurement | Source | Tags | Main Fields |
| --- | --- | --- | --- |
| `ping_metrics` | ICMP/TCP fallback | `device_id`, `workspace`, `ip` | `latency`, `packet_loss`, `response_time`, `status_up` |
| `tcp_metrics` | TCP worker | `device_id`, `workspace`, `ip`, `port` | `connect_duration`, `success`, `timeout`, `error` |
| `ap_metrics` | Ruijie API, SNMP | `device_id`, `workspace`, `ip`, `ap_name`, `source` | `client_count`, `cpu`, `memory`, `rssi`, `throughput`, `online`, `uptime` |
| `anomaly_metrics` | ML feature layer | `device_id`, `workspace`, `ip` | `score`, `latency_rolling_avg`, `packet_loss_ratio`, `ap_load_score`, `roaming_frequency`, `traffic_anomaly_score` |
| `syslog_events` | UDP syslog | `device_id`, `workspace`, `ip`, `facility`, `severity`, `hostname` | `message` |
