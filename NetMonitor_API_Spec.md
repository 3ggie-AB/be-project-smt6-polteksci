# NetMonitor API Specification

**Version:** v2.2  
**Base URL:** `http://localhost:8080`  
**Content-Type:** `application/json`  
**Auth:** Local email/password login + JWT Bearer token  
**Stack:** Go + Gin + MySQL + InfluxDB  

NetMonitor API dipakai untuk observability jaringan. Metadata relational disimpan di MySQL, sedangkan metrics time-series dari ping, TCP check, SNMP, Ruijie, syslog, dan anomaly disimpan di InfluxDB.

## Konsep Data

Mulai v2.2, metadata monitoring dipisah menjadi dua resource:

| Resource | Tabel | Tujuan |
| --- | --- | --- |
| Network Device | `devices` | Perangkat jaringan fisik/logis seperti router, switch, access point. Dipakai untuk SNMP dan Ruijie telemetry. |
| Monitoring Target | `monitoring_targets` | Target active monitoring seperti IP, domain, URL, atau server. Dipakai untuk ping dan TCP check. |

Dengan pemisahan ini, router/AP tidak perlu punya field `tcp_port`, `ping_interval_sec`, atau `tcp_interval_sec`. Ping dan TCP check dikelola lewat `/api/targets`.

## Authentication

Endpoint protected wajib mengirim:

```http
Authorization: Bearer <jwt_token>
```

Login pertama memakai `ADMIN_EMAIL` dan `ADMIN_PASSWORD` dari `.env` akan otomatis membuat user `SUPER_ADMIN` jika tabel user masih kosong.

Roles:

| Role | Hak Akses |
| --- | --- |
| `SUPER_ADMIN` | Full access |
| `ADMIN` | Mengelola device dan monitoring target |
| `USER` | Read-only dashboard, stream, notification, dan feature vector |

## Common Error Response

```json
{
  "error": "email atau password salah"
}
```

Status umum:

| Status | Arti |
| --- | --- |
| `400` | Request body atau parameter tidak valid |
| `401` | Token tidak ada atau login gagal |
| `403` | Role tidak cukup |
| `404` | Resource tidak ditemukan |
| `500` | Error internal backend |

## Endpoint Summary

| Method | Path | Auth | Role | Tujuan |
| --- | --- | --- | --- | --- |
| `GET` | `/` | No | Public | Info singkat API |
| `GET` | `/healthz` | No | Public | Health check komponen |
| `POST` | `/api/auth/login` | No | Public | Login dan mendapatkan JWT |
| `POST` | `/api/login` | No | Public | Alias login |
| `GET` | `/api/me` | Yes | All | Data user dari JWT |
| `GET` | `/api/stream` | Yes | All | SSE realtime event |
| `GET` | `/api/devices` | Yes | All | List router/AP/switch metadata |
| `POST` | `/api/devices` | Yes | `SUPER_ADMIN`, `ADMIN` | Tambah router/AP/switch |
| `DELETE` | `/api/devices/:id` | Yes | `SUPER_ADMIN`, `ADMIN` | Hapus device |
| `GET` | `/api/targets` | Yes | All | List ping/TCP monitoring target |
| `POST` | `/api/targets` | Yes | `SUPER_ADMIN`, `ADMIN` | Tambah target ping/TCP |
| `DELETE` | `/api/targets/:id` | Yes | `SUPER_ADMIN`, `ADMIN` | Hapus target |
| `GET` | `/api/notifications` | Yes | All | List unread notification |
| `POST` | `/api/notifications/:id/read` | Yes | All | Tandai notification read |
| `GET` | `/api/ml/features/:device_id` | Yes | All | Feature vector device/AP |
| `GET` | `/api/ml/features/targets/:target_id` | Yes | All | Feature vector ping/TCP target |

## Data Models

### User

```json
{
  "id": 1,
  "workspace_id": 1,
  "role_id": 1,
  "role": {
    "id": 1,
    "name": "SUPER_ADMIN"
  },
  "email": "admin@netmonitor.local",
  "name": "NetMonitor Admin",
  "avatar_url": "",
  "is_active": true,
  "last_login_at": "2026-05-10T13:40:00+07:00",
  "created_at": "2026-05-10T13:40:00+07:00",
  "updated_at": "2026-05-10T13:40:00+07:00"
}
```

### Network Device

Dipakai untuk router, switch, dan access point.

```json
{
  "id": 1,
  "workspace_id": 1,
  "name": "AP Lobby",
  "ip_address": "192.168.10.20",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "vendor": "ruijie",
  "model": "RG-AP820",
  "location": "Lobby",
  "device_type": "access_point",
  "snmp_version": "v2c",
  "ruijie_external_id": "ap-lobby-01",
  "is_active": true,
  "last_seen_at": "2026-05-10T13:45:00+07:00",
  "created_at": "2026-05-10T13:40:00+07:00",
  "updated_at": "2026-05-10T13:40:00+07:00"
}
```

`snmp_community` bisa dikirim saat create, tetapi tidak dikembalikan di response.

### Monitoring Target

Dipakai untuk ping dan TCP health check. Satu row hanya untuk satu jenis check.

```json
{
  "id": 10,
  "workspace_id": 1,
  "name": "Gateway Ping",
  "host": "192.168.1.1",
  "check_type": "ping",
  "port": 0,
  "interval_sec": 5,
  "timeout_sec": 3,
  "description": "Ping gateway utama",
  "is_active": true,
  "last_checked_at": "2026-05-10T13:55:00+07:00",
  "last_status": true,
  "created_at": "2026-05-10T13:50:00+07:00",
  "updated_at": "2026-05-10T13:50:00+07:00"
}
```

`check_type`:

| Value | Tujuan |
| --- | --- |
| `ping` | ICMP ping. `port` otomatis `0`. |
| `tcp` | TCP connect check. `port` wajib diisi, kecuali `host` dikirim sebagai URL dengan scheme `http`/`https`. |

### Realtime Event

```json
{
  "type": "tcp.service_down",
  "severity": "critical",
  "workspace": "default",
  "target_id": 11,
  "ip": "api.example.com",
  "title": "TCP service down",
  "message": "api.example.com:443 cannot be reached",
  "attributes": {
    "port": 443,
    "timeout": true,
    "error": "i/o timeout"
  },
  "occurred_at": "2026-05-10T13:55:00Z"
}
```

### Feature Vector

```json
{
  "device_id": 0,
  "target_id": 10,
  "workspace": "default",
  "latency_rolling_avg_ms": 32.5,
  "packet_loss_ratio": 0.01,
  "ap_load_score": 0,
  "roaming_frequency": 0,
  "traffic_anomaly_score": 0,
  "timestamp": "2026-05-10T13:55:00Z"
}
```

`onnx_input` order:

```text
[latency_rolling_avg_ms, packet_loss_ratio, ap_load_score, roaming_frequency, traffic_anomaly_score]
```

## Public Endpoints

### GET `/`

**Tujuan:** Info singkat bahwa service adalah NetMonitor API.

**Response 200**

```json
{
  "name": "NetMonitor API",
  "description": "Backend API untuk observability dan monitoring jaringan: ping, TCP health check, syslog, SNMP, Ruijie telemetry, realtime SSE, dan ML-ready metrics.",
  "status": "running",
  "endpoints": {
    "health": "/healthz",
    "login": "/api/auth/login",
    "stream": "/api/stream",
    "devices": "/api/devices",
    "targets": "/api/targets"
  }
}
```

### GET `/healthz`

**Tujuan:** Health check MySQL, InfluxDB, dan collector.

**Response 200**

```json
{
  "status": "ok",
  "time": "2026-05-10T06:45:00Z",
  "checks": {
    "mysql": {
      "status": "ok",
      "database": "netmonitor"
    },
    "influxdb": {
      "status": "enabled",
      "url": "http://localhost:8086",
      "bucket": "network_metrics"
    },
    "collectors": {
      "active": "enabled",
      "ruijie": "disabled",
      "syslog": "enabled",
      "snmp": "enabled"
    }
  }
}
```

### POST `/api/auth/login`

**Tujuan:** Login lokal dan mendapatkan JWT.

**Request**

```json
{
  "email": "admin@netmonitor.local",
  "password": "admin123"
}
```

**Response 200**

```json
{
  "message": "login berhasil",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "workspace_id": 1,
    "role": {
      "id": 1,
      "name": "SUPER_ADMIN"
    },
    "email": "admin@netmonitor.local",
    "name": "NetMonitor Admin",
    "is_active": true
  }
}
```

**Response 401**

```json
{
  "error": "email atau password salah"
}
```

## Protected Endpoints

### GET `/api/me`

**Tujuan:** Mengambil user context dari JWT.

**Response 200**

```json
{
  "user_id": 1,
  "email": "admin@netmonitor.local",
  "role": "SUPER_ADMIN",
  "workspace_id": 1
}
```

### GET `/api/stream`

**Tujuan:** Subscribe realtime monitoring event via SSE.

Header:

```http
Authorization: Bearer <jwt_token>
```

Atau query untuk browser `EventSource`:

```text
GET /api/stream?access_token=<jwt_token>
```

**SSE Example**

```text
event: latency.high
data: {"type":"latency.high","severity":"warning","workspace":"default","target_id":10,"ip":"192.168.1.1","title":"High latency","message":"Gateway Ping latency is 184.25 ms","attributes":{"latency_ms":184.25},"occurred_at":"2026-05-10T13:55:00Z"}
```

Event types:

| Type | Entity | Kapan Terjadi |
| --- | --- | --- |
| `ap.down` | device atau target | AP offline atau ping target down |
| `latency.high` | target | Latency melewati `HIGH_LATENCY_MS` |
| `packet_loss.high` | target | Packet loss melewati `HIGH_PACKET_LOSS_RATIO` |
| `tcp.service_down` | target | TCP connect gagal atau timeout |
| `anomaly.detected` | device atau target | Feature score melewati threshold |
| `syslog.alert` | device/IP | Syslog mengandung down/failed/critical/error |

## Devices API

### GET `/api/devices`

**Tujuan:** List router, switch, access point, atau perangkat jaringan lain.

**Response 200**

```json
[
  {
    "id": 1,
    "workspace_id": 1,
    "name": "AP Lobby",
    "ip_address": "192.168.10.20",
    "mac_address": "AA:BB:CC:DD:EE:FF",
    "vendor": "ruijie",
    "model": "RG-AP820",
    "location": "Lobby",
    "device_type": "access_point",
    "snmp_version": "v2c",
    "ruijie_external_id": "ap-lobby-01",
    "is_active": true,
    "last_seen_at": "2026-05-10T13:45:00+07:00",
    "created_at": "2026-05-10T13:40:00+07:00",
    "updated_at": "2026-05-10T13:40:00+07:00"
  }
]
```

### POST `/api/devices`

**Tujuan:** Tambah metadata perangkat jaringan untuk SNMP/Ruijie/passive telemetry.

**Role:** `SUPER_ADMIN`, `ADMIN`

Field:

| Field | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `name` | string | Yes | Nama device |
| `ip_address` | string | Yes | IP perangkat |
| `device_type` | string | No | `router`, `switch`, `access_point`, default `network` |
| `vendor` | string | No | Vendor perangkat |
| `model` | string | No | Model perangkat |
| `location` | string | No | Lokasi perangkat |
| `mac_address` | string | No | MAC address |
| `snmp_community` | string | No | Secret SNMP v2c |
| `snmp_version` | string | No | Default `v2c` |
| `ruijie_external_id` | string | No | Mapping ID dari Ruijie API |
| `is_active` | bool | No | Default `true` |

#### Contoh: Access Point Ruijie

```json
{
  "name": "AP Lobby",
  "ip_address": "192.168.10.20",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "vendor": "ruijie",
  "model": "RG-AP820",
  "device_type": "access_point",
  "location": "Lobby",
  "ruijie_external_id": "ap-lobby-01",
  "is_active": true
}
```

#### Contoh: Router/Switch dengan SNMP

```json
{
  "name": "Switch Core Lt 2",
  "ip_address": "192.168.10.2",
  "vendor": "cisco",
  "model": "CBS350",
  "device_type": "switch",
  "location": "Rack Lt 2",
  "snmp_community": "public",
  "snmp_version": "v2c",
  "is_active": true
}
```

**Response 201**

```json
{
  "id": 2,
  "workspace_id": 1,
  "name": "Switch Core Lt 2",
  "ip_address": "192.168.10.2",
  "vendor": "cisco",
  "model": "CBS350",
  "location": "Rack Lt 2",
  "device_type": "switch",
  "snmp_version": "v2c",
  "is_active": true,
  "created_at": "2026-05-10T13:51:00+07:00",
  "updated_at": "2026-05-10T13:51:00+07:00"
}
```

SNMP metrics akan masuk ke InfluxDB measurement `ap_metrics` dengan `source=snmp`.

### DELETE `/api/devices/:id`

**Tujuan:** Hapus device metadata.

**Response 200**

```json
{
  "message": "device deleted"
}
```

## Monitoring Targets API

### GET `/api/targets`

**Tujuan:** List target ping dan TCP active monitoring.

**Response 200**

```json
[
  {
    "id": 10,
    "workspace_id": 1,
    "name": "Gateway Ping",
    "host": "192.168.1.1",
    "check_type": "ping",
    "port": 0,
    "interval_sec": 5,
    "timeout_sec": 3,
    "description": "Ping gateway utama",
    "is_active": true,
    "last_checked_at": "2026-05-10T13:55:00+07:00",
    "last_status": true
  },
  {
    "id": 11,
    "workspace_id": 1,
    "name": "API Production HTTPS",
    "host": "api.example.com",
    "check_type": "tcp",
    "port": 443,
    "interval_sec": 30,
    "timeout_sec": 3,
    "description": "TCP check HTTPS API",
    "is_active": true,
    "last_checked_at": "2026-05-10T13:55:00+07:00",
    "last_status": true
  }
]
```

### POST `/api/targets`

**Tujuan:** Tambah target active monitoring. Ping dan TCP dipisah lewat `check_type`.

**Role:** `SUPER_ADMIN`, `ADMIN`

Field:

| Field | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `name` | string | Yes | Nama target |
| `host` | string | Yes | IP, hostname, domain, atau URL |
| `check_type` | string | No | `ping` atau `tcp`, default `ping` |
| `port` | int | Untuk TCP | Port TCP. Bisa otomatis dari URL `http/https`. |
| `interval_sec` | int | No | Default ping `5`, TCP `30` |
| `timeout_sec` | int | No | Default `3` |
| `description` | string | No | Catatan target |
| `is_active` | bool | No | Default `true` |

#### Contoh: Ping Target

**Tujuan:** Ping gateway atau IP server.

```json
{
  "name": "Gateway Ping",
  "host": "192.168.1.1",
  "check_type": "ping",
  "interval_sec": 5,
  "timeout_sec": 3,
  "description": "Ping gateway utama",
  "is_active": true
}
```

**Response 201**

```json
{
  "id": 10,
  "workspace_id": 1,
  "name": "Gateway Ping",
  "host": "192.168.1.1",
  "check_type": "ping",
  "port": 0,
  "interval_sec": 5,
  "timeout_sec": 3,
  "description": "Ping gateway utama",
  "is_active": true,
  "created_at": "2026-05-10T13:55:00+07:00",
  "updated_at": "2026-05-10T13:55:00+07:00"
}
```

InfluxDB:

| Measurement | Fields |
| --- | --- |
| `ping_metrics` | `latency`, `packet_loss`, `response_time`, `status_up` |

Realtime event:

- `latency.high`
- `packet_loss.high`
- `ap.down` untuk target down
- `anomaly.detected`

#### Contoh: TCP Target dengan Host + Port

**Tujuan:** Cek service TCP seperti SSH, HTTP, HTTPS, MySQL, atau API server.

```json
{
  "name": "API Production HTTPS",
  "host": "api.example.com",
  "check_type": "tcp",
  "port": 443,
  "interval_sec": 30,
  "timeout_sec": 3,
  "description": "TCP check HTTPS API",
  "is_active": true
}
```

#### Contoh: TCP Target dengan URL

**Tujuan:** Frontend boleh kirim URL. Backend akan menyimpan hostname dan port otomatis.

```json
{
  "name": "Landing Page HTTPS",
  "host": "https://example.com",
  "check_type": "tcp",
  "interval_sec": 30,
  "timeout_sec": 3
}
```

Response akan menjadi:

```json
{
  "id": 12,
  "workspace_id": 1,
  "name": "Landing Page HTTPS",
  "host": "example.com",
  "check_type": "tcp",
  "port": 443,
  "interval_sec": 30,
  "timeout_sec": 3,
  "is_active": true
}
```

InfluxDB:

| Measurement | Fields |
| --- | --- |
| `tcp_metrics` | `connect_duration`, `success`, `timeout`, `error` |

Realtime event:

- `tcp.service_down`

### DELETE `/api/targets/:id`

**Tujuan:** Hapus monitoring target.

**Response 200**

```json
{
  "message": "target deleted"
}
```

## Notification API

### GET `/api/notifications`

**Tujuan:** List unread notification.

**Response 200**

```json
[
  {
    "id": 10,
    "workspace_id": 1,
    "user_id": null,
    "device_id": null,
    "type": "tcp.service_down",
    "severity": "critical",
    "title": "TCP service down",
    "message": "api.example.com:443 cannot be reached",
    "read_at": null,
    "created_at": "2026-05-10T13:55:00+07:00"
  }
]
```

### POST `/api/notifications/:id/read`

**Response 200**

```json
{
  "message": "notification marked as read"
}
```

## ML Feature API

### GET `/api/ml/features/:device_id`

**Tujuan:** Feature vector untuk device/AP dari Ruijie/SNMP telemetry.

**Response 200**

```json
{
  "features": {
    "device_id": 1,
    "target_id": 0,
    "workspace": "default",
    "latency_rolling_avg_ms": 0,
    "packet_loss_ratio": 0,
    "ap_load_score": 0.42,
    "roaming_frequency": 0.08,
    "traffic_anomaly_score": 1.7,
    "timestamp": "2026-05-10T13:55:00Z"
  },
  "onnx_input": [0, 0, 0.42, 0.08, 1.7]
}
```

### GET `/api/ml/features/targets/:target_id`

**Tujuan:** Feature vector untuk ping/TCP target.

**Response 200**

```json
{
  "features": {
    "device_id": 0,
    "target_id": 10,
    "workspace": "default",
    "latency_rolling_avg_ms": 32.5,
    "packet_loss_ratio": 0.01,
    "ap_load_score": 0,
    "roaming_frequency": 0,
    "traffic_anomaly_score": 0,
    "timestamp": "2026-05-10T13:55:00Z"
  },
  "onnx_input": [32.5, 0.01, 0, 0, 0]
}
```

**Response 404**

```json
{
  "error": "feature vector not found"
}
```

## InfluxDB Measurement Strategy

### `ping_metrics`

Source: `monitoring_targets.check_type = ping`

Tags:

- `target_id`
- `workspace`
- `ip`

Fields:

- `latency`
- `packet_loss`
- `response_time`
- `status_up`

### `tcp_metrics`

Source: `monitoring_targets.check_type = tcp`

Tags:

- `target_id`
- `workspace`
- `ip`
- `port`

Fields:

- `connect_duration`
- `success`
- `timeout`
- `error`

### `ap_metrics`

Source: `devices` via Ruijie API or SNMP.

Tags:

- `device_id`
- `workspace`
- `ip`
- `ap_name`
- `source`

Fields:

- `client_count`
- `cpu`
- `memory`
- `rssi`
- `throughput`
- `online`
- `uptime`

### `anomaly_metrics`

Source: feature engineering layer.

Tags:

- `device_id` for device/AP features
- `target_id` for ping/TCP target features
- `workspace`
- `ip`

Fields:

- `score`
- `latency_rolling_avg`
- `packet_loss_ratio`
- `ap_load_score`
- `roaming_frequency`
- `traffic_anomaly_score`
- `model`

### `syslog_events`

Source: UDP syslog receiver.

Tags:

- `device_id`
- `workspace`
- `ip`
- `facility`
- `severity`
- `hostname`

Fields:

- `message`

## Example Flow

### 1. Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@netmonitor.local","password":"admin123"}'
```

### 2. Tambah Router/AP Device

```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AP Lobby",
    "ip_address": "192.168.10.20",
    "vendor": "ruijie",
    "device_type": "access_point",
    "ruijie_external_id": "ap-lobby-01",
    "is_active": true
  }'
```

### 3. Tambah Ping Target

```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gateway Ping",
    "host": "192.168.1.1",
    "check_type": "ping"
  }'
```

### 4. Tambah TCP Target

```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Production HTTPS",
    "host": "https://api.example.com",
    "check_type": "tcp"
  }'
```

### 5. Subscribe Realtime Stream

```js
const stream = new EventSource("http://localhost:8080/api/stream?access_token=<jwt_token>");

stream.addEventListener("latency.high", (event) => {
  console.log(JSON.parse(event.data));
});

stream.addEventListener("tcp.service_down", (event) => {
  console.log(JSON.parse(event.data));
});
```
