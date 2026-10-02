package database

import (
	"context"
	"database/sql"
	"time"
)

// Open is a thin wrapper; migrations live in backend/migrations and are
// applied via `ispctl migrate` (golang-migrate compatible, ordered SQL).
func Open(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func Ping(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
