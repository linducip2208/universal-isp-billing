# Deployment (aaPanel-friendly, no Docker required)

FRESH SERVER -> INSTALL -> CONFIGURE -> MIGRATE -> ADMIN -> START -> HEALTH -> READY:

1. PostgreSQL 14+ + Redis 6+ (or compatible). Create `isp` database.
2. `psql $DATABASE_URL -f backend/migrations/001_init.sql` then
   `002_seed.sql 003_rules.sql 004_constraints_indexes.sql 005_partitioning.sql
   006_password_policy.sql 007_operations.sql 008_demo_flag.sql 009_list_indexes.sql`
   (order matters; each file is idempotent via IF NOT EXISTS).
3. Build: `cd backend && go build -o /opt/isp/bin/server ./cmd/server &&
   go build -o /opt/isp/bin/ispctl ./cmd/ispctl`; frontend:
   `cd frontend && npm install && npm run build`, serve `dist/` via nginx
   (`deployments/nginx/isp.conf`).
4. Env (`deployments/env.example`): `DATABASE_URL`, `REDIS_ADDR`,
   `JWT_SECRET` (32+ chars), `SECRETS_KEY`, `ISP_DEMO_LOGIN` (leave unset),
   `CORS_ORIGINS`.
5. systemd: `isp-api.service`, `isp-worker.service` (+ scheduler via
   `ispctl scheduler` unit copy). `ispctl doctor` + `ispctl health` gates.
6. Create first admin directly in `users` (PBKDF2 hash via `ispctl`? PLANNED
   helper — today insert with application-generated hash), assign role.
7. TLS via certbot; log rotation via journald/logrotate examples (PLANNED file).
8. Backup: `ispctl backup` (pg_dump, verified 63KB round-trip); restore:
   `psql $DATABASE_URL < backup.sql`. RPO = backup frequency, RTO = restore
   + migrate time (documented, not automated).
9. Upgrade: pull, `go build`, apply new migrations in order, restart units.
   Rollback: down-migrations per file (PLANNED automation) or DB restore.
