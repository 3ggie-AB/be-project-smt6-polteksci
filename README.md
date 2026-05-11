# NetMonitor

Fiber + GORM + MySQL backend untuk monitoring jaringan dengan struktur direktori mirip Laravel.

## Struktur

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

## Run Locally

```bash
cp .env.example .env
go run .
```

Health check:

```bash
curl http://localhost:8080/healthz
```

## Auth

Login pertama akan membuat user pertama sebagai `SUPER_ADMIN`.

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"secret123"}'
```

## Resource API

Semua resource punya CRUD:

- `/api/users`
- `/api/sessions`
- `/api/devices`
- `/api/device-status`
- `/api/monitoring-configs`
- `/api/alerts`
- `/api/notifications`
- `/api/activity-logs`
- `/api/network-topology`
- `/api/ml-predictions`
- `/api/ml-anomalies`

Lihat [NetMonitor_API_Spec.md](NetMonitor_API_Spec.md) untuk request dan response lengkap.
