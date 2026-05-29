# NetMonitor API Implementation Specification

**Version:** v4.0  
**Base URL:** `http://localhost:8080`  
**Stack:** Go Fiber + GORM + MySQL  
**Auth:** Opaque session token via `Authorization: Bearer <token>`  

Dokumen ini mengikuti implementasi backend saat ini. Semua endpoint resource berada di bawah `/api` dan membutuhkan token login, kecuali root, health check, dan login.

## Runtime Behavior

Saat aplikasi start:

1. Load `.env`.
2. Buat database MySQL jika belum ada.
3. Jalankan GORM `AutoMigrate`.
4. Register route Fiber.
5. Start background monitoring worker untuk ping dan SNMP jika enabled.

Worker monitoring membaca tabel `devices` dan `monitoring_configs`, menjalankan probe, menyimpan hasil ke `device_status`, lalu mengubah `devices.status` menjadi `ONLINE`, `WARNING`, atau `OFFLINE`.

## Authentication

Login menghasilkan token random sepanjang 64 karakter. Token plaintext hanya dikirim sekali pada response login. Database menyimpan hash SHA-256 token di `sessions.token_hash`; field token/hash tidak muncul di response JSON session.

Header protected endpoint:

```http
Authorization: Bearer <token>
```

Query `access_token` masih diterima untuk kompatibilitas, tetapi header `Authorization` lebih disarankan.

### POST `/api/auth/login`

Alias: `POST /api/login`

Login pertama saat tabel `users` kosong akan membuat user pertama sebagai `SUPER_ADMIN`.

Request:

```json
{
  "email": "admin@example.com",
  "password": "secret123"
}
```

Success `200`:

```json
{
  "message": "login berhasil",
  "token": "SESSION_TOKEN",
  "session": {
    "id": 1,
    "user_id": 1,
    "expired_at": "2026-05-30T10:00:00+07:00",
    "created_at": "2026-05-29T10:00:00+07:00"
  },
  "user": {
    "id": 1,
    "name": "admin",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN",
    "created_at": "2026-05-29T10:00:00+07:00"
  }
}
```

Error `401`:

```json
{
  "error": "email or password is invalid"
}
```

### GET `/api/auth/me`

Alias: `GET /api/me`

Success `200`:

```json
{
  "data": {
    "id": 1,
    "name": "admin",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN",
    "created_at": "2026-05-29T10:00:00+07:00"
  }
}
```

### POST `/api/auth/logout`

Menghapus session berdasarkan hash token aktif.

Success `200`:

```json
{
  "message": "logout berhasil"
}
```

## Authorization Rules

| Area | SUPER_ADMIN | ADMIN | USER |
| --- | --- | --- | --- |
| Login, logout, me | Yes | Yes | Yes |
| List/detail protected resource | Yes | Yes | Yes |
| Create user | Yes | Yes, regular `USER` only | No |
| Update user | Yes | Regular `USER` only, cannot promote role | No |
| Delete user | Yes | No | No |
| Create/update/delete non-user resource | Yes | Yes | No |

`/api/sessions` tidak disediakan sebagai CRUD publik/protected untuk mencegah kebocoran token session.

## Root And Health

### GET `/`

Public endpoint untuk info aplikasi dan daftar endpoint utama.

Success `200`:

```json
{
  "name": "NetMonitor API",
  "description": "Fiber API untuk monitoring jaringan, device inventory, alerting, notification, topology, dan ML observability.",
  "status": "running",
  "endpoints": {
    "health": "/healthz",
    "login": "/api/auth/login",
    "users": "/api/users",
    "devices": "/api/devices",
    "device_status": "/api/device-status",
    "monitoring_configs": "/api/monitoring-configs",
    "alerts": "/api/alerts",
    "notifications": "/api/notifications",
    "activity_logs": "/api/activity-logs",
    "network_topology": "/api/network-topology",
    "ml_predictions": "/api/ml-predictions",
    "ml_anomalies": "/api/ml-anomalies"
  }
}
```

### GET `/healthz`

Public endpoint untuk health check sederhana.

Success `200`:

```json
{
  "status": "ok",
  "stack": "fiber + gorm + mysql",
  "mysql": {
    "host": "localhost",
    "port": "3306",
    "database": "netmonitor"
  }
}
```

## Standard CRUD Pattern

Semua resource di bawah ini memakai pola umum:

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| `GET` | `/api/<resource>` | Any logged-in user | List data |
| `POST` | `/api/<resource>` | `SUPER_ADMIN`, `ADMIN` | Create data |
| `GET` | `/api/<resource>/:id` | Any logged-in user | Detail data |
| `PUT` | `/api/<resource>/:id` | `SUPER_ADMIN`, `ADMIN` | Full/partial update |
| `PATCH` | `/api/<resource>/:id` | `SUPER_ADMIN`, `ADMIN` | Partial update |
| `DELETE` | `/api/<resource>/:id` | `SUPER_ADMIN`, `ADMIN` | Delete data |

List response:

```json
{
  "data": []
}
```

Create/detail/update response:

```json
{
  "data": {}
}
```

Delete response:

```json
{
  "message": "resource deleted"
}
```

## Users

Base path: `/api/users`

User delete hanya boleh oleh `SUPER_ADMIN`.

Create:

```json
{
  "name": "Operator",
  "email": "operator@example.com",
  "password": "secret123",
  "role": "USER"
}
```

Rules:

- `name`, `email`, dan `password` wajib saat create.
- Email harus valid.
- Password minimal 8 karakter.
- Role valid: `SUPER_ADMIN`, `ADMIN`, `USER`.
- `ADMIN` hanya boleh create/update regular `USER`.

Response:

```json
{
  "data": {
    "id": 2,
    "name": "Operator",
    "email": "operator@example.com",
    "role": "USER",
    "created_at": "2026-05-29T10:05:00+07:00"
  }
}
```

## Devices

Base path: `/api/devices`

Create:

```json
{
  "name": "AP Lobby",
  "ip": "192.168.10.20",
  "type": "AP",
  "vendor": "Ruijie",
  "location": "Lobby",
  "status": "OFFLINE"
}
```

Fields:

| Field | Type | Notes |
| --- | --- | --- |
| `name` | string | Required |
| `ip` | string | Device IP/host target for ping/SNMP |
| `type` | enum | `AP`, `SERVICE` |
| `vendor` | string | Optional |
| `location` | string | Optional |
| `status` | enum | `ONLINE`, `OFFLINE`, `WARNING` |

Monitoring worker akan memperbarui `status` berdasarkan hasil ping/SNMP.

## Monitoring Configs

Base path: `/api/monitoring-configs`

Create:

```json
{
  "device_id": 1,
  "ping_enabled": true,
  "tcp_enabled": false,
  "ping_interval": 5,
  "tcp_interval": 30,
  "monitored_port": 443
}
```

Current implementation:

- `ping_enabled` dipakai worker untuk menentukan apakah device diping.
- `tcp_enabled`, `tcp_interval`, dan `monitored_port` sudah tersimpan, tetapi TCP probe belum dijalankan oleh worker saat ini.
- Jika device belum punya config, worker memakai default ping enabled.

## Device Status

Base path: `/api/device-status`

Worker ping/SNMP otomatis menulis data ke resource ini. Endpoint CRUD tetap tersedia untuk admin.

Create manual:

```json
{
  "device_id": 1,
  "latency": 12.5,
  "packet_loss": 0,
  "cpu_usage": 35.2,
  "memory_usage": 61.8,
  "last_seen": "2026-05-29T10:12:00+07:00"
}
```

Fields:

| Field | Type | Notes |
| --- | --- | --- |
| `latency` | number | Average ping latency in ms |
| `packet_loss` | number | Packet loss percentage, e.g. `0`, `33.33`, `100` |
| `cpu_usage` | number | Value from configured SNMP CPU OID |
| `memory_usage` | number | Value from configured SNMP memory OID |
| `last_seen` | timestamp | Probe timestamp |

## Alerts

Base path: `/api/alerts`

Create:

```json
{
  "device_id": 1,
  "severity": "CRITICAL",
  "message": "AP Lobby offline",
  "status": "ACTIVE"
}
```

Enums:

- `severity`: `INFO`, `WARNING`, `CRITICAL`
- `status`: `ACTIVE`, `RESOLVED`

## Notifications

Base path: `/api/notifications`

Create:

```json
{
  "user_id": 1,
  "alert_id": 1,
  "title": "Critical Alert",
  "message": "AP Lobby offline",
  "is_read": false
}
```

## Activity Logs

Base path: `/api/activity-logs`

Create:

```json
{
  "user_id": 1,
  "action": "CREATE_DEVICE",
  "description": "Created AP Lobby device"
}
```

Catatan: pencatatan otomatis via middleware belum aktif; data saat ini dibuat melalui endpoint.

## Network Topology

Base path: `/api/network-topology`

Create:

```json
{
  "source_device_id": 1,
  "target_device_id": 2,
  "relation_type": "uplink",
  "status": "active"
}
```

## ML Predictions

Base path: `/api/ml-predictions`

Create:

```json
{
  "device_id": 1,
  "prediction_type": "latency_next_5m",
  "prediction_value": 32.7,
  "confidence_score": 0.91
}
```

## ML Anomalies

Base path: `/api/ml-anomalies`

Create:

```json
{
  "device_id": 1,
  "anomaly_score": 0.87,
  "prediction": "traffic spike detected",
  "severity": "WARNING"
}
```

Enums:

- `severity`: `WARNING`, `CRITICAL`

## Monitoring Worker Configuration

`.env` keys:

```env
MONITORING_ENABLED=true
PING_ENABLED=true
PING_INTERVAL=5s
PING_TIMEOUT=3s
PING_COUNT=3
PING_WORKERS=64
HIGH_LATENCY_MS=0
HIGH_PACKET_LOSS_RATIO=0

SNMP_ENABLED=false
SNMP_POLL_INTERVAL=60s
SNMP_PORT=161
SNMP_TIMEOUT=3s
SNMP_RETRIES=1
SNMP_COMMUNITY=public
SNMP_VERSION=2c
SNMP_CPU_OID=
SNMP_MEMORY_OID=
SNMP_WORKERS=64
```

Ping behavior:

- `packet_loss = 100` membuat device `OFFLINE`.
- Device reachable dengan packet loss atau latency melebihi threshold menjadi `WARNING`.
- Device reachable tanpa warning menjadi `ONLINE`.
- `HIGH_PACKET_LOSS_RATIO` menerima bentuk rasio `0.2` atau persentase `20`; keduanya diperlakukan sebagai 20%.
- Jika threshold bernilai `0`, setiap packet loss di atas 0 menjadi `WARNING`.

SNMP behavior:

- SNMP aktif hanya jika `SNMP_ENABLED=true` dan minimal salah satu dari `SNMP_CPU_OID` atau `SNMP_MEMORY_OID` terisi.
- `SNMP_VERSION` mendukung `1`/`v1`; nilai lain default ke `2c`.
- Nilai SNMP numerik disimpan ke `cpu_usage` dan `memory_usage`.

## Error Responses

Body JSON invalid:

```json
{
  "error": "invalid request body",
  "detail": "..."
}
```

Unauthorized:

```json
{
  "error": "authentication token is required"
}
```

Forbidden:

```json
{
  "error": "insufficient permission"
}
```

Not found:

```json
{
  "error": "device not found"
}
```

Database/server error:

```json
{
  "error": "failed to create device",
  "detail": "..."
}
```

## Curl Flow

Login:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"secret123"}' | jq -r .token)
```

Create device:

```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Localhost","ip":"127.0.0.1","type":"SERVICE","vendor":"Local","location":"Lab","status":"OFFLINE"}'
```

Create monitoring config:

```bash
curl -X POST http://localhost:8080/api/monitoring-configs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":1,"ping_enabled":true,"tcp_enabled":false,"ping_interval":5,"tcp_interval":30,"monitored_port":0}'
```

Read latest status:

```bash
curl http://localhost:8080/api/device-status \
  -H "Authorization: Bearer $TOKEN"
```

Logout:

```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```
