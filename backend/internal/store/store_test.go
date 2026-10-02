package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
)

func TestNoDatabaseHonest(t *testing.T) {
	var s *store.Store
	if _, err := s.List(context.Background(), "o1", "customers", store.Query{}); err != store.ErrNoDatabase {
		t.Fatalf("nil store must return ErrNoDatabase, got %v", err)
	}
	s2 := store.New(nil)
	if _, err := s2.GetByID(context.Background(), "o1", "customers", "x"); err != store.ErrNoDatabase {
		t.Fatalf("nil db must return ErrNoDatabase, got %v", err)
	}
	if _, err := s2.List(context.Background(), "o1", "nope", store.Query{}); err != store.ErrNoDatabase {
		t.Fatalf("want ErrNoDatabase first, got %v", err)
	}
}

// TestLivePostgres runs against a real database when TEST_DATABASE_URL is
// set (CI). It is skipped otherwise — unit CI never needs credentials.
func TestLivePostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := store.New(db)
	pg, err := s.List(context.Background(), "11111111-1111-1111-1111-111111111111", "customers", store.Query{PerPage: 5})
	if err != nil {
		t.Fatal(err)
	}
	if pg.Page != 1 || pg.PerPage != 5 {
		t.Fatalf("bad page %+v", pg)
	}
}
