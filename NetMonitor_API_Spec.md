# NetMonitor API Specification

**Version:** v2.1  
**Base URL:** `http://localhost:8080`  
**Content-Type:** `application/json`  
**Auth:** Local email/password login + JWT Bearer token  
**Stack:** Go + Gin + MySQL + InfluxDB  

NetMonitor API adalah backend untuk observability dan monitoring jaringan. API ini mengelola user, role, device metadata, realtime event stream, notification, dan ML-ready feature vector. Data time-series seperti ping, TCP check, SNMP, Ruijie telemetry, syslog, dan anomaly metrics disimpan di InfluxDB oleh collector backend.

## Authentication

Endpoint protected wajib mengirim header:

```http
Authorization: Bearer <jwt_token>
```

Saat database user masih kosong, login pertama memakai `ADMIN_EMAIL` dan `ADMIN_PASSWORD` dari `.env` akan otomatis membuat user dengan role `SUPER_ADMIN`.

Roles:

| Role | Keterangan |
| --- | --- |
| `SUPER_ADMIN` | Akses penuh semua operasi admin |
| `ADMIN` | Bisa mengelola device dan monitoring metadata |
| `USER` | Read-only dashboard, stream, dan feature view |

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
| `401` | Token tidak ada, token invalid, atau login gagal |
| `403` | Role tidak cukup |
| `404` | Resource tidak ditemukan |
| `500` | Error internal backend |

## Data Models

### User

```json
{
  "id": 1,
  "workspace_id": 1,
  "workspace": {
    "id": 1,
    "name": "Default Workspace",
    "slug": "default",
    "created_at": "2026-05-10T13:00:00+07:00",
    "updated_at": "2026-05-10T13:00:00+07:00"
  },
  "role_id": 1,
  "role": {
    "id": 1,
    "name": "SUPER_ADMIN",
    "created_at": "2026-05-10T13:00:00+07:00",
    "updated_at": "2026-05-10T13:00:00+07:00"
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

### Device

```json
{
  "id": 1,
  "workspace_id": 1,
  "workspace": {
    "id": 1,
    "name": "Default Workspace",
    "slug": "default"
  },
  "name": "AP Lobby",
  "ip_address": "192.168.10.20",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "vendor": "ruijie",
  "model": "RG-AP820",
  "location": "Lobby",
  "device_type": "ap",
  "tcp_port": 443,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "snmp_version": "v2c",
  "ruijie_external_id": "ap-lobby-01",
  "is_active": true,
  "last_seen_at": "2026-05-10T13:45:00+07:00",
  "created_at": "2026-05-10T13:40:00+07:00",
  "updated_at": "2026-05-10T13:40:00+07:00"
}
```

`snmp_community` tidak dikembalikan di response karena termasuk secret.

### Notification

```json
{
  "id": 10,
  "workspace_id": 1,
  "user_id": null,
  "device_id": 1,
  "type": "latency.high",
  "severity": "warning",
  "title": "High latency",
  "message": "AP Lobby latency is 184.25 ms",
  "read_at": null,
  "created_at": "2026-05-10T13:45:00+07:00"
}
```

### Realtime Event

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
  "occurred_at": "2026-05-10T13:45:00Z"
}
```

### Feature Vector

```json
{
  "device_id": 1,
  "workspace": "default",
  "latency_rolling_avg_ms": 32.5,
  "packet_loss_ratio": 0.01,
  "ap_load_score": 0.42,
  "roaming_frequency": 0.08,
  "traffic_anomaly_score": 1.7,
  "timestamp": "2026-05-10T13:45:00Z"
}
```

`onnx_input` order:

```text
[latency_rolling_avg_ms, packet_loss_ratio, ap_load_score, roaming_frequency, traffic_anomaly_score]
```

## Endpoint Summary

| Method | Path | Auth | Role | Tujuan |
| --- | --- | --- | --- | --- |
| `GET` | `/` | No | Public | Info singkat API |
| `GET` | `/healthz` | No | Public | Health check dan status komponen |
| `POST` | `/api/auth/login` | No | Public | Login lokal dan mendapatkan JWT |
| `POST` | `/api/login` | No | Public | Alias login untuk kompatibilitas |
| `GET` | `/api/me` | Yes | All | Melihat identity user dari token |
| `GET` | `/api/stream` | Yes | All | Subscribe realtime event via SSE |
| `GET` | `/api/devices` | Yes | All | List device metadata |
| `POST` | `/api/devices` | Yes | `SUPER_ADMIN`, `ADMIN` | Register device untuk ping/TCP/SNMP/Ruijie |
| `DELETE` | `/api/devices/:id` | Yes | `SUPER_ADMIN`, `ADMIN` | Hapus device |
| `GET` | `/api/notifications` | Yes | All | List unread notification |
| `POST` | `/api/notifications/:id/read` | Yes | All | Tandai notification sebagai read |
| `GET` | `/api/ml/features/:device_id` | Yes | All | Ambil feature vector untuk ML/ONNX |

## Public Endpoints

### GET `/`

**Tujuan:** Menampilkan informasi singkat bahwa service yang sedang berjalan adalah NetMonitor API.

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
    "devices": "/api/devices"
  }
}
```

### GET `/healthz`

**Tujuan:** Health check untuk dashboard, uptime monitor, Docker/Kubernetes probe, atau manual debugging.

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

**Tujuan:** Login dengan email/password dan mendapatkan JWT.

**Request Body**

```json
{
  "email": "admin@netmonitor.local",
  "password": "admin123"
}
```

Field:

| Field | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `email` | string | Optional | Jika kosong, backend memakai `ADMIN_EMAIL` dari `.env` |
| `password` | string | Yes | Password user atau `ADMIN_PASSWORD` untuk bootstrap pertama |

**Response 200**

```json
{
  "message": "login berhasil",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "workspace_id": 1,
    "role_id": 1,
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

### POST `/api/login`

**Tujuan:** Alias untuk `/api/auth/login`. Response sama.

## Authenticated Endpoints

### GET `/api/me`

**Tujuan:** Mengambil identity user berdasarkan JWT aktif.

**Headers**

```http
Authorization: Bearer <jwt_token>
```

**Response 200**

```json
{
  "user_id": 1,
  "email": "admin@netmonitor.local",
  "role": "SUPER_ADMIN",
  "workspace_id": 1
}
```

**Response 401**

```json
{
  "error": "authentication token is required"
}
```

### GET `/api/stream`

**Tujuan:** Subscribe realtime monitoring events via Server-Sent Events.

Dipakai dashboard untuk event:

- AP down
- latency tinggi
- packet loss tinggi
- TCP service down
- anomaly detection
- syslog alert

**Auth Options**

Header:

```http
Authorization: Bearer <jwt_token>
```

Atau query untuk browser `EventSource`:

```text
GET /api/stream?access_token=<jwt_token>
```

**SSE Event Example**

```text
event: latency.high
data: {"type":"latency.high","severity":"warning","workspace":"default","device_id":1,"ip":"192.168.10.20","title":"High latency","message":"AP Lobby latency is 184.25 ms","attributes":{"latency_ms":184.25},"occurred_at":"2026-05-10T13:45:00Z"}
```

**Heartbeat Example**

```text
event: heartbeat
data: {"ts":"2026-05-10T13:45:15Z"}
```

### GET `/api/devices`

**Tujuan:** Mengambil daftar device metadata yang dipantau.

**Response 200**

```json
[
  {
    "id": 1,
    "workspace_id": 1,
    "workspace": {
      "id": 1,
      "name": "Default Workspace",
      "slug": "default"
    },
    "name": "AP Lobby",
    "ip_address": "192.168.10.20",
    "mac_address": "AA:BB:CC:DD:EE:FF",
    "vendor": "ruijie",
    "model": "RG-AP820",
    "location": "Lobby",
    "device_type": "ap",
    "tcp_port": 443,
    "ping_interval_sec": 5,
    "tcp_interval_sec": 30,
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

**Tujuan:** Register device supaya collector backend bisa menjalankan ping, TCP health check, SNMP polling, atau mapping Ruijie telemetry.

**Role:** `SUPER_ADMIN`, `ADMIN`

Field penting:

| Field | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `name` | string | Yes | Nama device |
| `ip_address` | string | Yes | IP device/AP/router/switch |
| `vendor` | string | No | Contoh: `ruijie`, `mikrotik`, `cisco` |
| `device_type` | string | No | Contoh: `ap`, `router`, `switch`, default `network` |
| `tcp_port` | int | No | Port TCP health check, default `443` |
| `ping_interval_sec` | int | No | Interval ping per device, default `5` |
| `tcp_interval_sec` | int | No | Interval TCP check per device, default `30` |
| `snmp_community` | string | No | Dibutuhkan untuk SNMP v2c polling |
| `snmp_version` | string | No | Default `v2c` |
| `ruijie_external_id` | string | No | ID mapping dari Ruijie API |
| `is_active` | bool | No | Default `true` |

#### Contoh 1: Device untuk Ping + TCP Check

**Tujuan:** Memantau gateway dengan active monitoring. Backend akan menjalankan ping tiap 5 detik dan TCP check port 80 tiap 30 detik.

```json
{
  "name": "Gateway Utama",
  "ip_address": "192.168.1.1",
  "vendor": "mikrotik",
  "device_type": "router",
  "location": "Ruang Server",
  "tcp_port": 80,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "is_active": true
}
```

**Response 201**

```json
{
  "id": 1,
  "workspace_id": 1,
  "name": "Gateway Utama",
  "ip_address": "192.168.1.1",
  "vendor": "mikrotik",
  "location": "Ruang Server",
  "device_type": "router",
  "tcp_port": 80,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "snmp_version": "v2c",
  "is_active": true,
  "created_at": "2026-05-10T13:50:00+07:00",
  "updated_at": "2026-05-10T13:50:00+07:00"
}
```

Data yang akan masuk ke InfluxDB:

| Measurement | Isi |
| --- | --- |
| `ping_metrics` | latency, packet loss, response time, status up/down |
| `tcp_metrics` | connect duration, success/fail, timeout |

Realtime event yang mungkin muncul:

- `latency.high`
- `packet_loss.high`
- `ap.down` / device down
- `tcp.service_down`

#### Contoh 2: Device untuk SNMP Collector

**Tujuan:** Memantau switch/AP/router via SNMP. Backend akan polling SNMP jika `SNMP_ENABLED=true` dan device punya `snmp_community`.

```json
{
  "name": "Switch Core Lt 2",
  "ip_address": "192.168.10.2",
  "vendor": "cisco",
  "model": "CBS350",
  "device_type": "switch",
  "location": "Rack Lt 2",
  "tcp_port": 22,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
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
  "tcp_port": 22,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "snmp_version": "v2c",
  "is_active": true,
  "created_at": "2026-05-10T13:51:00+07:00",
  "updated_at": "2026-05-10T13:51:00+07:00"
}
```

Data yang akan masuk ke InfluxDB:

| Measurement | Isi |
| --- | --- |
| `ap_metrics` | CPU, memory, uptime, total interface traffic sebagai throughput, online status |
| `anomaly_metrics` | traffic anomaly score jika feature layer mendeteksi pola aneh |

Catatan SNMP:

- `snmp_community` tidak muncul di response.
- OID CPU dan memory vendor-specific bisa diset via `.env`: `SNMP_CPU_OID`, `SNMP_MEMORY_OID`.
- Interface traffic memakai IF-MIB octets.

#### Contoh 3: Ruijie AP Mapping

**Tujuan:** Metadata device untuk menghubungkan AP lokal dengan telemetry dari Ruijie API collector.

```json
{
  "name": "AP Lobby",
  "ip_address": "192.168.10.20",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "vendor": "ruijie",
  "model": "RG-AP820",
  "device_type": "ap",
  "location": "Lobby",
  "tcp_port": 443,
  "ping_interval_sec": 5,
  "tcp_interval_sec": 30,
  "ruijie_external_id": "ap-lobby-01",
  "is_active": true
}
```

Ruijie collector akan menyimpan telemetry ke `ap_metrics`:

- AP name
- AP IP
- client count
- CPU
- memory
- RSSI
- throughput
- online status

### DELETE `/api/devices/:id`

**Tujuan:** Menghapus device metadata.

**Role:** `SUPER_ADMIN`, `ADMIN`

**Path Parameter**

| Parameter | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `id` | integer | Yes | ID device |

**Response 200**

```json
{
  "message": "device deleted"
}
```

**Response 400**

```json
{
  "error": "id must be a positive integer"
}
```

### GET `/api/notifications`

**Tujuan:** Mengambil unread notification untuk workspace aktif.

**Response 200**

```json
[
  {
    "id": 10,
    "workspace_id": 1,
    "user_id": null,
    "device_id": 1,
    "type": "tcp.service_down",
    "severity": "critical",
    "title": "TCP service down",
    "message": "192.168.1.1:80 cannot be reached",
    "read_at": null,
    "created_at": "2026-05-10T13:55:00+07:00"
  }
]
```

### POST `/api/notifications/:id/read`

**Tujuan:** Menandai notification sebagai sudah dibaca.

**Response 200**

```json
{
  "message": "notification marked as read"
}
```

### GET `/api/ml/features/:device_id`

**Tujuan:** Mengambil feature vector terbaru dari in-memory feature engineering layer untuk device tertentu.

**Path Parameter**

| Parameter | Type | Required | Keterangan |
| --- | --- | --- | --- |
| `device_id` | integer | Yes | ID device |

**Response 200**

```json
{
  "features": {
    "device_id": 1,
    "workspace": "default",
    "latency_rolling_avg_ms": 32.5,
    "packet_loss_ratio": 0.01,
    "ap_load_score": 0.42,
    "roaming_frequency": 0.08,
    "traffic_anomaly_score": 1.7,
    "timestamp": "2026-05-10T13:55:00Z"
  },
  "onnx_input": [32.5, 0.01, 0.42, 0.08, 1.7]
}
```

**Response 404**

```json
{
  "error": "feature vector not found"
}
```

## Realtime Event Types

| Type | Severity | Kapan Terjadi |
| --- | --- | --- |
| `ap.down` | `critical` | Device/AP tidak merespon ping atau telemetry menunjukkan offline |
| `latency.high` | `warning` | Latency melewati `HIGH_LATENCY_MS` |
| `packet_loss.high` | `warning` | Packet loss melewati `HIGH_PACKET_LOSS_RATIO` |
| `tcp.service_down` | `critical` | TCP port gagal connect atau timeout |
| `anomaly.detected` | `warning` | Feature score melewati threshold |
| `syslog.alert` | sesuai syslog | Syslog mengandung indikasi down/failed/critical/error |

## Internal InfluxDB Measurement Strategy

Bagian ini bukan HTTP API, tapi kontrak data time-series yang dipakai collector backend.

### `ping_metrics`

Source: active monitoring ping worker.

Tags:

| Tag | Contoh |
| --- | --- |
| `device_id` | `1` |
| `workspace` | `default` |
| `ip` | `192.168.1.1` |

Fields:

| Field | Type | Keterangan |
| --- | --- | --- |
| `latency` | float | Average RTT dalam ms |
| `packet_loss` | float | Packet loss persen `0-100` |
| `response_time` | float | Total response time probe dalam ms |
| `status_up` | bool | `true` jika reachable |

Example point:

```json
{
  "measurement": "ping_metrics",
  "tags": {
    "device_id": "1",
    "workspace": "default",
    "ip": "192.168.1.1"
  },
  "fields": {
    "latency": 12.34,
    "packet_loss": 0,
    "response_time": 15.2,
    "status_up": true
  }
}
```

### `tcp_metrics`

Source: active monitoring TCP worker.

Tags:

- `device_id`
- `workspace`
- `ip`
- `port`

Fields:

- `connect_duration`
- `success`
- `timeout`
- `error`

### `ap_metrics`

Source: Ruijie API and SNMP collector.

Tags:

- `device_id`
- `workspace`
- `ip`
- `ap_name`
- `source`: `ruijie_api` or `snmp`

Fields:

- `client_count`
- `cpu`
- `memory`
- `rssi`
- `throughput`
- `online`
- `uptime`

### `anomaly_metrics`

Source: ML feature engineering layer.

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

## Example Frontend Flow

### 1. Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@netmonitor.local","password":"admin123"}'
```

### 2. Register Device untuk Ping

```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gateway Utama",
    "ip_address": "192.168.1.1",
    "device_type": "router",
    "tcp_port": 80,
    "ping_interval_sec": 5,
    "tcp_interval_sec": 30,
    "is_active": true
  }'
```

### 3. Register Device untuk SNMP

```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Switch Core Lt 2",
    "ip_address": "192.168.10.2",
    "vendor": "cisco",
    "device_type": "switch",
    "tcp_port": 22,
    "snmp_community": "public",
    "snmp_version": "v2c",
    "is_active": true
  }'
```

### 4. Subscribe Realtime Stream

```js
const stream = new EventSource("http://localhost:8080/api/stream?access_token=<jwt_token>");

stream.addEventListener("latency.high", (event) => {
  console.log(JSON.parse(event.data));
});

stream.addEventListener("tcp.service_down", (event) => {
  console.log(JSON.parse(event.data));
});
```

### 5. Read ML Feature Vector

```bash
curl http://localhost:8080/api/ml/features/1 \
  -H "Authorization: Bearer <jwt_token>"
```
