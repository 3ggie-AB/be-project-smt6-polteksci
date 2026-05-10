# NetMonitor Enterprise Observability Architecture

## Folder Structure

```text
cmd/server/           HTTP server entrypoint
internal/app/         Dependency injection, router wiring, graceful shutdown
internal/config/      Environment config loader
domain/              Entity and metric contracts
repository/          Repository interfaces
mysql/               GORM MySQL store and migrations
influx/              Async InfluxDB writer with batching and retry
auth/                Local password login and JWT service
middleware/          JWT and RBAC middleware
service/             Application services
collector/           Active and passive telemetry collectors
ml/                  Feature engineering and ONNX-ready vectors
websocket/           SSE realtime broker
migrations/          SQL schema
```

## Persistence Strategy

MySQL is authoritative for metadata:

- `users`, `roles`, `workspaces`
- `devices`, `device_groups`, `device_group_members`
- `monitoring_targets`
- `notifications`

InfluxDB is authoritative for time-series:

- `ping_metrics`
- `tcp_metrics`
- `ap_metrics`
- `anomaly_metrics`
- `syslog_events`

Influx tags:

- `device_id`
- `target_id`
- `workspace`
- `ip`
- `ap_name` when available
- `source` for passive telemetry

Influx fields:

- ping: `latency`, `packet_loss`, `response_time`, `status_up`
- TCP: `connect_duration`, `success`, `timeout`, `error`
- AP/SNMP/Ruijie: `client_count`, `cpu`, `memory`, `rssi`, `throughput`, `online`, `uptime`
- anomaly: rolling feature values plus `score`
- syslog: `message`

Raw Ruijie JSON is intentionally not persisted. It is logged only when `RUIJIE_DEBUG_RAW_JSON=true`.

## Authentication

Local login is exposed at:

- `POST /api/auth/login`
- `POST /api/login` for compatibility with older clients

If no user exists yet, the first successful login using `ADMIN_EMAIL` and `ADMIN_PASSWORD` becomes `SUPER_ADMIN`. Later users default to `USER` when created by admin workflows.
JWT claims include `user_id`, `email`, `role`, and `workspace_id`; RBAC middleware protects admin write routes.

## Monitoring Pipeline

Active monitoring:

- Ping reads rows from `monitoring_targets` where `check_type=ping`
- TCP check reads rows from `monitoring_targets` where `check_type=tcp`
- Ping scheduler interval defaults to `PING_INTERVAL=5s`
- TCP scheduler interval defaults to `TCP_INTERVAL=30s`
- Worker pools are bounded by `PING_WORKERS` and `TCP_WORKERS`
- Metrics are written to InfluxDB asynchronously
- Realtime events are published to SSE

Passive monitoring:

- Ruijie API polling writes AP metrics
- SNMP polling writes AP/device telemetry fields
- UDP Syslog receiver writes log events

## Realtime Streaming

Dashboard clients can subscribe through:

```text
GET /api/stream?access_token=<jwt>
```

Events include:

- `ap.down`
- `latency.high`
- `packet_loss.high`
- `tcp.service_down`
- `anomaly.detected`
- `syslog.alert`

## Machine Learning Preparation

The feature layer produces an ONNX-ready vector:

```text
[latency_rolling_avg_ms, packet_loss_ratio, ap_load_score, roaming_frequency, traffic_anomaly_score]
```

The vector is suitable as a feature contract for:

- Isolation Forest
- LSTM
- Graph Neural Network node features
- ONNX inference runtimes

Endpoint:

```text
GET /api/ml/features/:device_id
GET /api/ml/features/targets/:target_id
```
