# Development

Backend: `cd backend && go test ./... && go vet ./... && go build ./...`.
DB: `psql $DATABASE_URL -f backend/migrations/001_init.sql`.
Frontend: `cd frontend && npm install && npm run dev`.
Conventions: context-first funcs, structured slog, money in cents,
idempotency keys on all network mutations.
