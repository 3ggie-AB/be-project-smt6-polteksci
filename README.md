# Network Monitor API

Sistem monitoring jaringan perusahaan berbasis **Go Fiber**, **MySQL**, dan **ClickHouse**.

---

## Tech Stack

| Komponen | Teknologi |
|---|---|
| Framework | Go Fiber v2 |
| Database Relasional | MySQL + GORM |
| Database Time-series | ClickHouse |
| Auth | JWT (HS256) |
| ICMP Ping | go-ping/ping |
| SNMP | gosnmp |
| Password | bcrypt |

---

## Persyaratan

- Go 1.21+
- MySQL 8.0+
- ClickHouse 23+

---

## Instalasi & Menjalankan

```bash
cd go-server

# Salin dan sesuaikan konfigurasi
cp .env.example .env
nano .env

# Unduh dependency
go mod tidy

# Jalankan server
go run main.go
```

---

## Konfigurasi ENV

| Variabel | Default | Keterangan |
|---|---|---|
| `APP_PORT` | `8080` | Port server |
| `APP_ENV` | `development` | Mode environment |
| `JWT_SECRET` | — | Kunci rahasia JWT |
| `JWT_EXPIRE_HOURS` | `24` | Masa berlaku token (jam) |
| `MYSQL_HOST` | `localhost` | Host MySQL |
| `MYSQL_PORT` | `3306` | Port MySQL |
| `MYSQL_USER` | `root` | User MySQL |
| `MYSQL_PASSWORD` | — | Password MySQL |
| `MYSQL_DATABASE` | `network_monitor` | Nama database MySQL |
| `CLICKHOUSE_HOST` | `localhost` | Host ClickHouse |
| `CLICKHOUSE_PORT` | `9000` | Port native ClickHouse |
| `CLICKHOUSE_USER` | `default` | User ClickHouse |
| `CLICKHOUSE_PASSWORD` | — | Password ClickHouse |
| `CLICKHOUSE_DATABASE` | `network_monitor` | Nama database ClickHouse |
| `ADMIN_EMAIL` | `admin@company.com` | Email akun atasan |
| `ADMIN_PASSWORD` | `Admin@1234` | Password akun atasan |
| `TEKNISI_EMAIL` | `teknisi@company.com` | Email akun teknisi IT |
| `TEKNISI_PASSWORD` | `Teknisi@1234` | Password akun teknisi |
| `STAFF_EMAIL` | `staff@company.com` | Email akun staff |
| `STAFF_PASSWORD` | `Staff@1234` | Password akun staff |
| `KARYAWAN_EMAIL` | `karyawan@company.com` | Email akun karyawan |
| `KARYAWAN_PASSWORD` | `Karyawan@1234` | Password akun karyawan |

---

## Default Data (Seeder)

Saat pertama kali dijalankan, sistem otomatis membuat:

### Akun Default

| Nama | Email | Password | Role |
|---|---|---|---|
| Administrator | admin@company.com | Admin@1234 | Atasan |
| Budi Santoso | teknisi@company.com | Teknisi@1234 | Teknisi IT |
| Siti Rahayu | staff@company.com | Staff@1234 | Staff |
| Agus Pratama | karyawan@company.com | Karyawan@1234 | Karyawan |

### Perangkat Default

| Nama | IP | Tipe |
|---|---|---|
| Router Utama | 192.168.1.1 | router |
| Switch Core | 192.168.1.254 | switch |
| Server Web | 192.168.1.10 | server |
| Server Database | 192.168.1.11 | server |
| Firewall UTM | 192.168.1.2 | firewall |
| Access Point Lantai 2 | 192.168.2.1 | access_point |
| Google DNS | 8.8.8.8 | other |

### Feedback/Keluhan Default

6 contoh feedback dengan berbagai status, kategori, dan prioritas.

---

## RBAC — Hak Akses per Role

| Fitur | Atasan | Teknisi IT | Staff | Karyawan |
|---|:---:|:---:|:---:|:---:|
| Lihat User | ✅ | ✅ | ✗ | ✗ |
| Buat/Edit User | ✅ | ✗ | ✗ | ✗ |
| Hapus User | ✅ | ✗ | ✗ | ✗ |
| Reset Password | ✅ | ✗ | ✗ | ✗ |
| Kelola Role & Permission | ✅ | ✗ | ✗ | ✗ |
| Lihat Perangkat | ✅ | ✅ | ✅ | ✗ |
| Tambah/Edit Perangkat | ✅ | ✅ | ✗ | ✗ |
| Hapus Perangkat | ✅ | ✗ | ✗ | ✗ |
| Jalankan Ping / SNMP | ✅ | ✅ | ✗ | ✗ |
| Lihat Histori Monitoring | ✅ | ✅ | ✅ | ✗ |
| Lihat Semua Feedback | ✅ | ✅ | ✅ | ✗ |
| Buat Feedback | ✅ | ✅ | ✅ | ✅ |
| Balas Feedback | ✅ | ✅ | ✗ | ✗ |
| Hapus Feedback | ✅ | ✗ | ✗ | ✗ |

---

## Endpoint API

Base URL: `http://localhost:8080/api/v1`

### Autentikasi

```
POST /auth/login           Login, mendapat JWT token
POST /auth/register        Daftar akun baru (role: karyawan)
GET  /auth/me              Profil user yang sedang login
PUT  /auth/change-password Ganti password sendiri
```

**Contoh Login:**
```json
POST /api/v1/auth/login
{
    "email": "admin@company.com",
    "password": "Admin@1234"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Login berhasil",
    "data": {
        "token": "eyJhbGci...",
        "expire_in": 86400,
        "user": { "id": 1, "name": "Administrator", "role": {...} },
        "permissions": ["users:read", "users:write", ...]
    }
}
```

Tambahkan header di semua request selanjutnya:
```
Authorization: Bearer <token>
```

---

### Manajemen User

```
GET    /users                  Daftar user (atasan, teknisi)
POST   /users                  Buat user baru (atasan)
GET    /users/:id              Detail user
PUT    /users/:id              Update user
DELETE /users/:id              Hapus user (atasan)
POST   /users/:id/reset-password  Reset password (atasan)
```

**Query params GET /users:**
- `page`, `limit`, `search`, `role`

---

### Role & Permission

```
GET  /roles                     Daftar role beserta permission
GET  /roles/:id                 Detail role
PUT  /roles/:id/permissions     Update permission role (atasan)
GET  /permissions               Semua permission tersedia
```

---

### Perangkat Jaringan

```
GET    /devices                 Daftar perangkat
POST   /devices                 Tambah perangkat
GET    /devices/:id             Detail perangkat
PUT    /devices/:id             Update perangkat
DELETE /devices/:id             Hapus perangkat
```

**Contoh Tambah Perangkat:**
```json
POST /api/v1/devices
{
    "name": "Server Backup",
    "ip_address": "192.168.1.20",
    "type": "server",
    "location": "Server Room Lt. 2",
    "description": "Server backup harian",
    "snmp_community": "public",
    "snmp_version": "2c",
    "snmp_port": 161
}
```

---

### Monitoring (Ping & SNMP)

```
POST /monitoring/ping                        Ping IP custom
POST /monitoring/devices/:id/ping            Ping perangkat terdaftar
GET  /monitoring/devices/:id/ping/history    Histori ping (ClickHouse)
POST /monitoring/devices/:id/snmp            SNMP GET perangkat
GET  /monitoring/devices/:id/snmp/history    Histori SNMP (ClickHouse)
GET  /monitoring/oids                        Daftar OID umum tersedia
```

**Contoh Ping:**
```json
POST /api/v1/monitoring/ping
{
    "ip_address": "8.8.8.8",
    "count": 4
}
```

**Response Ping:**
```json
{
    "success": true,
    "data": {
        "ip_address": "8.8.8.8",
        "packets_sent": 4,
        "packets_received": 4,
        "packet_loss": 0,
        "min_rtt_ms": 12.5,
        "avg_rtt_ms": 14.2,
        "max_rtt_ms": 18.1,
        "status": "up"
    }
}
```

**Contoh SNMP:**
```json
POST /api/v1/monitoring/devices/1/snmp
{
    "oids": ["1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.5.0"]
}
```
*(Kosongkan `oids` untuk mengambil semua OID umum)*

---

### Feedback / Keluhan

```
GET  /feedback                Daftar feedback
GET  /feedback/stats          Statistik feedback
POST /feedback                Buat feedback baru
GET  /feedback/:id            Detail feedback
PUT  /feedback/:id            Update feedback
POST /feedback/:id/respond    Balas feedback (teknisi/atasan)
DELETE /feedback/:id          Hapus feedback (atasan)
```

**Query params GET /feedback:**
- `page`, `limit`, `search`, `status`, `category`, `priority`, `my_only=true`

**Kategori:** `keluhan`, `saran`, `pertanyaan`, `insiden`
**Status:** `open`, `in_progress`, `resolved`, `closed`
**Prioritas:** `low`, `medium`, `high`, `critical`

**Contoh Buat Feedback:**
```json
POST /api/v1/feedback
{
    "title": "Koneksi VPN Terputus Saat WFH",
    "description": "Setiap pagi hari sekitar jam 08.00-09.00 VPN sering disconnect. Sudah coba reconnect tapi tetap tidak stabil.",
    "category": "keluhan",
    "priority": "high"
}
```

**Contoh Balas Feedback:**
```json
POST /api/v1/feedback/1/respond
{
    "response": "Kami sudah identifikasi masalah pada server VPN. Akan diperbaiki dalam 2 jam ke depan.",
    "status": "in_progress"
}
```

---

## Struktur Proyek

```
go-server/
├── main.go                          Entry point
├── go.mod / go.sum                  Go modules
├── .env.example                     Contoh konfigurasi
└── internal/
    ├── config/config.go             Konfigurasi dari ENV
    ├── database/
    │   ├── mysql.go                 Koneksi MySQL + GORM
    │   └── clickhouse.go           Koneksi & migrasi ClickHouse
    ├── models/
    │   ├── user.go                  Model User, Role, Permission
    │   ├── device.go                Model Device/Perangkat
    │   └── feedback.go              Model Feedback/Keluhan
    ├── middleware/
    │   ├── auth.go                  JWT middleware & helper
    │   └── rbac.go                  RBAC permission check
    ├── handlers/
    │   ├── auth.go                  Login, Register, Me
    │   ├── user.go                  CRUD User & Role
    │   ├── monitoring.go            Ping, SNMP, Device CRUD
    │   └── feedback.go              CRUD Feedback & Respond
    ├── services/
    │   ├── ping.go                  ICMP ping + simpan ke ClickHouse
    │   └── snmp.go                  SNMP GET + simpan ke ClickHouse
    ├── routes/routes.go             Registrasi semua route
    └── seeder/seeder.go             Data default (users, devices, feedback)
```
