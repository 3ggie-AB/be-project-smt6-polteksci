# NetMonitor

Enterprise-grade network observability backend built with Go, MySQL, InfluxDB, JWT local login, active monitoring, passive collectors, SSE realtime streaming, and ML-ready feature engineering.

## Run Locally

```bash
cp .env.example .env
docker compose up -d mysql influxdb
go run ./cmd/server
```

API health check:

```bash
curl http://localhost:8080/healthz
```

## Key Endpoints

- `POST /api/auth/login`
- `GET /api/devices`
- `POST /api/devices`
- `GET /api/targets`
- `POST /api/targets`
- `GET /api/stream?access_token=<jwt>`
- `GET /api/ml/features/:device_id`
- `GET /api/ml/features/targets/:target_id`

## Architecture

See [docs/observability_architecture.md](docs/observability_architecture.md).
