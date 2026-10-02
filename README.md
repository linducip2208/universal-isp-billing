# Universal ISP Billing & Network Automation Platform

Production-grade, vendor-neutral ISP billing, subscriber management, network
automation, provisioning, monitoring, and NOC platform (Go + React).

## Quickstart (Windows dev)

```powershell
# DB
psql $env:DATABASE_URL -f backend/migrations/001_init.sql
psql $env:DATABASE_URL -f backend/migrations/002_seed.sql
# Backend
cd backend; go test ./...; go run ./cmd/server
# Frontend
cd frontend; npm install; npm run dev
```

Login: `admin` / `secret` (demo only; production uses `users` table + bcrypt).

## Layout

- `backend/` Go modular monolith (Clean Architecture, zero external deps)
- `frontend/` React+TS+Vite, i18n en/id, dark/light
- `backend/migrations/` PostgreSQL schema + seed
- `backend/openapi/openapi.yaml` API v1
- `deployments/` systemd, nginx, env examples
- `docs/` full documentation + per-connector docs

## Key rules

- Money = int64 minor units. Every network action idempotent (idempotency keys).
- Billing depends only on `connectors/sdk.NetworkConnector`.
- No fabricated integrations: cloud vendors are marked REQUIRES_VENDOR_ACCESS
  until credentials + documented API tests exist (see VENDOR-MATRIX.md).
