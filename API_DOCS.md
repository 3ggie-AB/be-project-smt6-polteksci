# Network Monitor API Docs

Dokumentasi endpoint untuk backend Network Monitor berbasis Go Fiber.

## Base URL

```text
http://localhost:8080/api/v1
```

Root endpoint tersedia di:

```http
GET /
```

Endpoint ini mengembalikan daftar nama route, method, dan path API.

## Format Response

Response sukses umumnya memakai format:

```json
{
  "success": true,
  "message": "Pesan operasi",
  "data": {}
}
```

Response error umumnya memakai format:

```json
{
  "success": false,
  "message": "Pesan error"
}
```

Endpoint list yang memiliki pagination memakai `meta`:

```json
{
  "success": true,
  "data": [],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 10,
    "pages": 1
  }
}
```

## Authentication

Sebagian besar endpoint membutuhkan JWT Bearer token.

Header:

```http
Authorization: Bearer <token>
Content-Type: application/json
```

Token didapat dari endpoint login.

Role bawaan:

| Role | Keterangan |
|---|---|
| `atasan` | Akses penuh |
| `teknisi_it` | Monitoring, perangkat, feedback response, baca user |
| `staff` | Monitoring read, perangkat read, feedback |
| `karyawan` | Membuat feedback |

Permission bawaan:

| Permission | Keterangan |
|---|---|
| `users:read` | Lihat user |
| `users:write` | Buat/edit user |
| `users:delete` | Hapus user |
| `roles:read` | Lihat role/permission |
| `roles:write` | Update permission role |
| `monitoring:read` | Lihat histori monitoring |
| `monitoring:write` | Jalankan ping/SNMP |
| `devices:read` | Lihat perangkat |
| `devices:write` | Buat/edit perangkat |
| `devices:delete` | Hapus perangkat |
| `feedback:read` | Lihat feedback |
| `feedback:write` | Buat/edit feedback |
| `feedback:respond` | Balas feedback |
| `feedback:delete` | Hapus feedback |

## Endpoint Summary

| Name | Method | Route | Auth |
|---|---|---|---|
| Route Index | GET | `/` | No |
| Health Check | GET | `/api/v1/health` | No |
| Login | POST | `/api/v1/auth/login` | No |
| Register | POST | `/api/v1/auth/register` | No |
| Profil Saya | GET | `/api/v1/auth/me` | Yes |
| Ganti Password | PUT | `/api/v1/auth/change-password` | Yes |
| Daftar User | GET | `/api/v1/users` | Yes |
| Buat User | POST | `/api/v1/users` | Yes |
| Detail User | GET | `/api/v1/users/:id` | Yes |
| Update User | PUT | `/api/v1/users/:id` | Yes |
| Hapus User | DELETE | `/api/v1/users/:id` | Yes |
| Reset Password User | POST | `/api/v1/users/:id/reset-password` | Yes |
| Daftar Role | GET | `/api/v1/roles` | Yes |
| Detail Role | GET | `/api/v1/roles/:id` | Yes |
| Update Permission Role | PUT | `/api/v1/roles/:id/permissions` | Yes |
| Daftar Permission | GET | `/api/v1/permissions` | Yes |
| Daftar Perangkat | GET | `/api/v1/devices` | Yes |
| Tambah Perangkat | POST | `/api/v1/devices` | Yes |
| Detail Perangkat | GET | `/api/v1/devices/:id` | Yes |
| Update Perangkat | PUT | `/api/v1/devices/:id` | Yes |
| Hapus Perangkat | DELETE | `/api/v1/devices/:id` | Yes |
| Ping Custom | POST | `/api/v1/monitoring/ping` | Yes |
| Ping Perangkat | POST | `/api/v1/monitoring/devices/:id/ping` | Yes |
| Histori Ping Perangkat | GET | `/api/v1/monitoring/devices/:id/ping/history` | Yes |
| SNMP Perangkat | POST | `/api/v1/monitoring/devices/:id/snmp` | Yes |
| Histori SNMP Perangkat | GET | `/api/v1/monitoring/devices/:id/snmp/history` | Yes |
| Daftar OID | GET | `/api/v1/monitoring/oids` | Yes |
| Daftar Feedback | GET | `/api/v1/feedback` | Yes |
| Statistik Feedback | GET | `/api/v1/feedback/stats` | Yes |
| Buat Feedback | POST | `/api/v1/feedback` | Yes |
| Detail Feedback | GET | `/api/v1/feedback/:id` | Yes |
| Update Feedback | PUT | `/api/v1/feedback/:id` | Yes |
| Hapus Feedback | DELETE | `/api/v1/feedback/:id` | Yes |
| Balas Feedback | POST | `/api/v1/feedback/:id/respond` | Yes |

## Public Endpoints

### Route Index

```http
GET /
```

Response:

```json
{
  "success": true,
  "message": "Daftar route Network Monitor API",
  "base_url": "/api/v1",
  "docs": "API_DOCS.md",
  "data": [
    {
      "name": "Login",
      "method": "POST",
      "route": "/api/v1/auth/login"
    }
  ]
}
```

### Health Check

```http
GET /api/v1/health
```

Response:

```json
{
  "success": true,
  "message": "Network Monitor API berjalan dengan normal",
  "version": "1.0.0"
}
```

## Auth

### Login

```http
POST /api/v1/auth/login
```

Body:

```json
{
  "email": "admin@admin.com",
  "password": "admin"
}
```

Response:

```json
{
  "success": true,
  "message": "Login berhasil",
  "data": {
    "token": "<jwt-token>",
    "expire_in": 86400,
    "user": {
      "id": 1,
      "name": "admin",
      "email": "admin@admin.com",
      "role_id": 1,
      "is_active": true
    },
    "permissions": ["users:read", "users:write"]
  }
}
```

### Register

User baru otomatis mendapat role `karyawan`.

```http
POST /api/v1/auth/register
```

Body:

```json
{
  "name": "Budi",
  "email": "budi@example.com",
  "password": "password123",
  "phone": "08123456789",
  "department": "IT"
}
```

### Profil Saya

```http
GET /api/v1/auth/me
```

Auth: Bearer token.

### Ganti Password

```http
PUT /api/v1/auth/change-password
```

Auth: Bearer token.

Body:

```json
{
  "old_password": "password123",
  "new_password": "password456"
}
```

## User Management

### Daftar User

```http
GET /api/v1/users?page=1&limit=10&search=admin&role=atasan
```

Auth: `users:read`.

Query:

| Query | Default | Keterangan |
|---|---|---|
| `page` | `1` | Halaman |
| `limit` | `10` | Jumlah data, maksimal 100 |
| `search` | kosong | Cari nama/email |
| `role` | kosong | Filter role, contoh `atasan` |

### Buat User

```http
POST /api/v1/users
```

Auth: `users:write`.

Body:

```json
{
  "name": "Teknisi Baru",
  "email": "teknisi.baru@example.com",
  "password": "password123",
  "role_id": 2,
  "phone": "08123456789",
  "department": "IT",
  "is_active": true
}
```

### Detail User

```http
GET /api/v1/users/:id
```

Auth: `users:read`.

### Update User

```http
PUT /api/v1/users/:id
```

Auth: `users:write`.

Body:

```json
{
  "name": "Nama Baru",
  "phone": "08123456789",
  "department": "IT",
  "role_id": 2,
  "is_active": true
}
```

Catatan: role selain `atasan` hanya bisa mengedit profil sendiri dan tidak bisa mengubah `role_id` atau `is_active`.

### Hapus User

```http
DELETE /api/v1/users/:id
```

Auth: `users:delete`.

### Reset Password User

```http
POST /api/v1/users/:id/reset-password
```

Auth: role `atasan`.

Body:

```json
{
  "new_password": "passwordbaru123"
}
```

## Role & Permission

### Daftar Role

```http
GET /api/v1/roles
```

Auth: `roles:read`.

### Detail Role

```http
GET /api/v1/roles/:id
```

Auth: `roles:read`.

### Update Permission Role

```http
PUT /api/v1/roles/:id/permissions
```

Auth: `roles:write`.

Body:

```json
{
  "permission_ids": [1, 2, 3, 4]
}
```

### Daftar Permission

```http
GET /api/v1/permissions
```

Auth: `roles:read`.

## Devices

Device type yang disarankan: `router`, `switch`, `server`, `firewall`, `access_point`, `other`.

### Daftar Perangkat

```http
GET /api/v1/devices?page=1&limit=20&search=server&type=server
```

Auth: `devices:read`.

Query:

| Query | Default | Keterangan |
|---|---|---|
| `page` | `1` | Halaman |
| `limit` | `20` | Jumlah data, maksimal 100 |
| `search` | kosong | Cari nama/IP/lokasi |
| `type` | kosong | Filter tipe perangkat |

### Tambah Perangkat

```http
POST /api/v1/devices
```

Auth: `devices:write`.

Body:

```json
{
  "name": "Server Backup",
  "ip_address": "192.168.1.20",
  "type": "server",
  "location": "Server Room",
  "description": "Server backup harian",
  "snmp_community": "public",
  "snmp_version": "2c",
  "snmp_port": 161,
  "is_active": true
}
```

### Detail Perangkat

```http
GET /api/v1/devices/:id
```

Auth: `devices:read`.

### Update Perangkat

```http
PUT /api/v1/devices/:id
```

Auth: `devices:write`.

Body:

```json
{
  "name": "Server Backup Baru",
  "ip_address": "192.168.1.21",
  "type": "server",
  "location": "Server Room Lt. 2",
  "description": "Updated",
  "snmp_community": "public",
  "snmp_version": "2c",
  "snmp_port": 161,
  "is_active": true
}
```

### Hapus Perangkat

```http
DELETE /api/v1/devices/:id
```

Auth: `devices:delete`.

## Monitoring

Data histori ping dan SNMP disimpan ke ClickHouse.

### Ping Custom

```http
POST /api/v1/monitoring/ping
```

Auth: `monitoring:write`.

Body:

```json
{
  "ip_address": "8.8.8.8",
  "count": 4
}
```

Response:

```json
{
  "success": true,
  "message": "Ping selesai",
  "data": {
    "device_id": 0,
    "device_name": "Custom",
    "ip_address": "8.8.8.8",
    "packets_sent": 4,
    "packets_received": 4,
    "packet_loss": 0,
    "min_rtt_ms": 10.1,
    "avg_rtt_ms": 12.3,
    "max_rtt_ms": 15.4,
    "status": "up"
  }
}
```

### Ping Perangkat

```http
POST /api/v1/monitoring/devices/:id/ping
```

Auth: `monitoring:write`.

Body:

```json
{
  "count": 4
}
```

### Histori Ping Perangkat

```http
GET /api/v1/monitoring/devices/:id/ping/history?limit=50
```

Auth: `monitoring:read`.

Query:

| Query | Default | Keterangan |
|---|---|---|
| `limit` | `50` | Jumlah histori, maksimal 500 |

### SNMP Perangkat

```http
POST /api/v1/monitoring/devices/:id/snmp
```

Auth: `monitoring:write`.

Body:

```json
{
  "oids": [
    "1.3.6.1.2.1.1.1.0",
    "1.3.6.1.2.1.1.5.0"
  ]
}
```

Jika `oids` kosong, sistem memakai daftar OID bawaan.

### Histori SNMP Perangkat

```http
GET /api/v1/monitoring/devices/:id/snmp/history?limit=50
```

Auth: `monitoring:read`.

### Daftar OID

```http
GET /api/v1/monitoring/oids
```

Auth: `monitoring:read`.

## Feedback

Status: `open`, `in_progress`, `resolved`, `closed`.

Priority: `low`, `medium`, `high`, `critical`.

Category: `keluhan`, `saran`, `pertanyaan`, `insiden`.

### Daftar Feedback

```http
GET /api/v1/feedback?page=1&limit=10&status=open&category=keluhan&priority=high&search=internet&my_only=false
```

Auth: Bearer token.

Query:

| Query | Default | Keterangan |
|---|---|---|
| `page` | `1` | Halaman |
| `limit` | `10` | Jumlah data, maksimal 100 |
| `status` | kosong | Filter status |
| `category` | kosong | Filter kategori |
| `priority` | kosong | Filter prioritas |
| `search` | kosong | Cari title/description |
| `my_only` | `false` | Jika `true`, tampilkan feedback milik user login |

Catatan: role `karyawan` otomatis hanya melihat feedback miliknya sendiri.

### Statistik Feedback

```http
GET /api/v1/feedback/stats
```

Auth: Bearer token.

### Buat Feedback

```http
POST /api/v1/feedback
```

Auth: Bearer token.

Body:

```json
{
  "title": "Internet lambat",
  "description": "Koneksi internet lantai 2 lambat sejak pagi.",
  "category": "keluhan",
  "priority": "high"
}
```

### Detail Feedback

```http
GET /api/v1/feedback/:id
```

Auth: Bearer token.

Catatan: role `karyawan` hanya bisa melihat feedback miliknya sendiri.

### Update Feedback

```http
PUT /api/v1/feedback/:id
```

Auth: Bearer token.

Body:

```json
{
  "title": "Internet lambat lantai 2",
  "description": "Koneksi masih lambat.",
  "category": "keluhan",
  "priority": "critical",
  "status": "in_progress",
  "assigned_to_id": 2
}
```

Catatan: role `karyawan` tidak bisa mengubah feedback yang sudah `resolved` atau `closed`, dan tidak bisa mengubah `priority`, `status`, atau `assigned_to_id`.

### Hapus Feedback

```http
DELETE /api/v1/feedback/:id
```

Auth: `feedback:delete`.

### Balas Feedback

```http
POST /api/v1/feedback/:id/respond
```

Auth: `feedback:respond`.

Body:

```json
{
  "response": "Masalah sudah dicek dan sedang ditangani.",
  "status": "in_progress"
}
```

Jika `status` kosong, sistem otomatis mengubah status menjadi `resolved`.

## Status Code

| Code | Keterangan |
|---|---|
| `200` | Request berhasil |
| `201` | Resource berhasil dibuat |
| `400` | Request tidak valid |
| `401` | Token tidak ada/tidak valid |
| `403` | Role atau permission tidak cukup |
| `404` | Resource tidak ditemukan |
| `409` | Data konflik, contoh email/IP sudah terdaftar |
| `500` | Error server |

## cURL Singkat

Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@admin.com","password":"admin"}'
```

Ambil daftar perangkat:

```bash
curl http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer <token>"
```

Ping custom:

```bash
curl -X POST http://localhost:8080/api/v1/monitoring/ping \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"ip_address":"8.8.8.8","count":4}'
```
