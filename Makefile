.PHONY: build test test-race lint vet migrate seed run worker frontend

build:
	cd backend && go build ./...

test:
	cd backend && go test ./...

# Requires cgo/gcc (CI Linux). Not runnable on stock Windows.
test-race:
	cd backend && go test -race ./...

vet:
	cd backend && go vet ./...

fmt:
	cd backend && gofmt -l cmd internal

run:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run build

seed:
	@echo "psql $$DATABASE_URL -f backend/migrations/001_init.sql -f backend/migrations/002_seed.sql"

migrate:
	@echo "psql $$DATABASE_URL -f backend/migrations/001_init.sql -f backend/migrations/004_constraints_indexes.sql"
