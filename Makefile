.PHONY: build test lint vet migrate seed run worker

build:
	cd backend && go build ./...

test:
	cd backend && go test ./...

vet:
	cd backend && go vet ./...

run:
	cd backend && go run ./cmd/server

seed:
	@echo "psql $$DATABASE_URL -f backend/migrations/001_init.sql -f backend/migrations/002_seed.sql"
