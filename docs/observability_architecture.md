# NetMonitor Fiber Architecture

Project sekarang memakai Fiber + GORM + MySQL dengan struktur mirip Laravel.

```text
app/models              GORM models
app/http/controllers    HTTP controllers
app/http/middleware     Auth/role middleware
bootstrap               App bootstrapping
config                  Env config dan logger
database                MySQL connection + auto migrate
routes                  Route registration
migrations              SQL reference schema
```

## Tables

- `users`
- `sessions`
- `devices`
- `device_status`
- `monitoring_configs`
- `alerts`
- `notifications`
- `activity_logs`
- `network_topology`
- `ml_predictions`
- `ml_anomalies`

## Runtime

1. Load `.env`
2. Auto-create MySQL database jika belum ada
3. Connect MySQL
4. Run GORM `AutoMigrate`
5. Start Fiber server

## Auth

Auth memakai tabel `sessions`. Login membuat session token, lalu endpoint protected memakai:

```http
Authorization: Bearer <token>
```

User pertama yang login ketika tabel `users` kosong otomatis menjadi `SUPER_ADMIN`.

## API

Semua tabel punya CRUD API. Dokumentasi request dan response lengkap ada di [../NetMonitor_API_Spec.md](../NetMonitor_API_Spec.md).
