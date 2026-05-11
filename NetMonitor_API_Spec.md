# NetMonitor Fiber API Specification

**Version:** v3.0  
**Base URL:** `http://localhost:8080`  
**Stack:** Fiber + GORM + MySQL  
**Auth:** Session token via `Authorization: Bearer <token>`  

Backend sekarang memakai struktur mirip Laravel:

```text
app/models
app/http/controllers
app/http/middleware
bootstrap
config
database
routes
migrations
```

GORM menjalankan `AutoMigrate` untuk semua model saat aplikasi start.

## Authentication

Login pertama otomatis membuat user pertama sebagai `SUPER_ADMIN` jika tabel `users` masih kosong.

### POST `/api/auth/login`

Request:

```json
{
  "email": "admin@example.com",
  "password": "secret123"
}
```

Response:

```json
{
  "message": "login berhasil",
  "token": "SESSION_TOKEN",
  "session": {
    "id": 1,
    "user_id": 1,
    "token": "SESSION_TOKEN",
    "expired_at": "2026-05-12T10:00:00+07:00",
    "created_at": "2026-05-11T10:00:00+07:00"
  },
  "user": {
    "id": 1,
    "name": "admin",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN",
    "created_at": "2026-05-11T10:00:00+07:00"
  }
}
```

Gunakan token:

```http
Authorization: Bearer SESSION_TOKEN
```

### GET `/api/auth/me`

Response:

```json
{
  "data": {
    "id": 1,
    "name": "admin",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN",
    "created_at": "2026-05-11T10:00:00+07:00"
  }
}
```

### POST `/api/auth/logout`

Response:

```json
{
  "message": "logout berhasil"
}
```

## Standard CRUD Pattern

Semua tabel punya pola endpoint:

| Method | URL | Fungsi |
| --- | --- | --- |
| `GET` | `/api/<resource>` | List semua data |
| `POST` | `/api/<resource>` | Create data |
| `GET` | `/api/<resource>/:id` | Detail data |
| `PUT` | `/api/<resource>/:id` | Update data |
| `PATCH` | `/api/<resource>/:id` | Partial update |
| `DELETE` | `/api/<resource>/:id` | Delete data |

Response list:

```json
{
  "data": []
}
```

Response detail/create/update:

```json
{
  "data": {}
}
```

Error:

```json
{
  "error": "failed to create device",
  "detail": "..."
}
```

## Resource Endpoints

| Table | Resource URL |
| --- | --- |
| `users` | `/api/users` |
| `sessions` | `/api/sessions` |
| `devices` | `/api/devices` |
| `device_status` | `/api/device-status` |
| `monitoring_configs` | `/api/monitoring-configs` |
| `alerts` | `/api/alerts` |
| `notifications` | `/api/notifications` |
| `activity_logs` | `/api/activity-logs` |
| `network_topology` | `/api/network-topology` |
| `ml_predictions` | `/api/ml-predictions` |
| `ml_anomalies` | `/api/ml-anomalies` |

## Models And Examples

### Users

Table:

```dbml
Table users {
  id bigint [pk, increment]
  name varchar
  email varchar [unique]
  password varchar
  role varchar
  created_at timestamp
}
```

Create:

```http
POST /api/users
```

```json
{
  "name": "Operator",
  "email": "operator@example.com",
  "password": "secret123",
  "role": "USER"
}
```

Response:

```json
{
  "data": {
    "id": 2,
    "name": "Operator",
    "email": "operator@example.com",
    "role": "USER",
    "created_at": "2026-05-11T10:05:00+07:00"
  }
}
```

### Sessions

Table:

```dbml
Table sessions {
  id bigint [pk, increment]
  user_id bigint
  token text
  expired_at timestamp
  created_at timestamp
}
```

List:

```http
GET /api/sessions
```

Response:

```json
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "token": "SESSION_TOKEN",
      "expired_at": "2026-05-12T10:00:00+07:00",
      "created_at": "2026-05-11T10:00:00+07:00"
    }
  ]
}
```

### Devices

Table:

```dbml
Table devices {
  id bigint [pk, increment]
  name varchar
  ip varchar
  type enum("AP", "SERVICE")
  vendor varchar
  location varchar
  status enum("ONLINE", "OFFLINE", "WARNING")
  created_at timestamp
}
```

Create AP:

```http
POST /api/devices
```

```json
{
  "name": "AP Lobby",
  "ip": "192.168.10.20",
  "type": "AP",
  "vendor": "Ruijie",
  "location": "Lobby",
  "status": "ONLINE"
}
```

Create service:

```json
{
  "name": "API Production",
  "ip": "10.10.10.5",
  "type": "SERVICE",
  "vendor": "Internal",
  "location": "Data Center",
  "status": "WARNING"
}
```

Response:

```json
{
  "data": {
    "id": 1,
    "name": "AP Lobby",
    "ip": "192.168.10.20",
    "type": "AP",
    "vendor": "Ruijie",
    "location": "Lobby",
    "status": "ONLINE",
    "created_at": "2026-05-11T10:10:00+07:00"
  }
}
```

### Device Status

Table:

```dbml
Table device_status {
  id bigint [pk, increment]
  device_id bigint
  latency float
  packet_loss float
  cpu_usage float
  memory_usage float
  last_seen timestamp
}
```

Create:

```http
POST /api/device-status
```

```json
{
  "device_id": 1,
  "latency": 12.5,
  "packet_loss": 0.2,
  "cpu_usage": 40.5,
  "memory_usage": 55.1,
  "last_seen": "2026-05-11T10:12:00+07:00"
}
```

### Monitoring Configs

Table:

```dbml
Table monitoring_configs {
  id bigint [pk, increment]
  device_id bigint
  ping_enabled boolean
  tcp_enabled boolean
  ping_interval int
  tcp_interval int
  monitored_port int
  created_at timestamp
}
```

Create:

```http
POST /api/monitoring-configs
```

```json
{
  "device_id": 1,
  "ping_enabled": true,
  "tcp_enabled": true,
  "ping_interval": 5,
  "tcp_interval": 30,
  "monitored_port": 443
}
```

### Alerts

Table:

```dbml
Table alerts {
  id bigint [pk, increment]
  device_id bigint
  severity enum("INFO", "WARNING", "CRITICAL")
  message text
  status enum("ACTIVE", "RESOLVED")
  created_at timestamp
}
```

Create:

```http
POST /api/alerts
```

```json
{
  "device_id": 1,
  "severity": "CRITICAL",
  "message": "AP Lobby offline",
  "status": "ACTIVE"
}
```

### Notifications

Table:

```dbml
Table notifications {
  id bigint [pk, increment]
  user_id bigint
  alert_id bigint
  title varchar
  message text
  is_read boolean
  created_at timestamp
}
```

Create:

```http
POST /api/notifications
```

```json
{
  "user_id": 1,
  "alert_id": 1,
  "title": "Critical Alert",
  "message": "AP Lobby offline",
  "is_read": false
}
```

### Activity Logs

Table:

```dbml
Table activity_logs {
  id bigint [pk, increment]
  user_id bigint
  action varchar
  description text
  created_at timestamp
}
```

Create:

```http
POST /api/activity-logs
```

```json
{
  "user_id": 1,
  "action": "CREATE_DEVICE",
  "description": "Created AP Lobby device"
}
```

### Network Topology

Table:

```dbml
Table network_topology {
  id bigint [pk, increment]
  source_device_id bigint
  target_device_id bigint
  relation_type varchar
  status varchar
  created_at timestamp
}
```

Create:

```http
POST /api/network-topology
```

```json
{
  "source_device_id": 1,
  "target_device_id": 2,
  "relation_type": "uplink",
  "status": "active"
}
```

### ML Predictions

Table:

```dbml
Table ml_predictions {
  id bigint [pk, increment]
  device_id bigint
  prediction_type varchar
  prediction_value float
  confidence_score float
  created_at timestamp
}
```

Create:

```http
POST /api/ml-predictions
```

```json
{
  "device_id": 1,
  "prediction_type": "latency_next_5m",
  "prediction_value": 32.7,
  "confidence_score": 0.91
}
```

### ML Anomalies

Table:

```dbml
Table ml_anomalies {
  id bigint [pk, increment]
  device_id bigint
  anomaly_score float
  prediction varchar
  severity enum("WARNING", "CRITICAL")
  created_at timestamp
}
```

Create:

```http
POST /api/ml-anomalies
```

```json
{
  "device_id": 1,
  "anomaly_score": 0.87,
  "prediction": "traffic spike detected",
  "severity": "WARNING"
}
```

## Curl Flow Example

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"secret123"}' | jq -r .token)

curl http://localhost:8080/api/devices \
  -H "Authorization: Bearer $TOKEN"

curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"AP Lobby","ip":"192.168.10.20","type":"AP","vendor":"Ruijie","location":"Lobby","status":"ONLINE"}'
```
