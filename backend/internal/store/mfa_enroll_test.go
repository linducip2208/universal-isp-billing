package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/mfa"
	"github.com/universal-isp/platform/internal/security"
	"github.com/universal-isp/platform/internal/store"
)

func liveStore(t *testing.T) (*store.Store, context.Context, func()) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := security.DeriveKey("test-vault enroll")
	box, _ := security.NewSecretsBox(key)
	s := store.New(db).WithSecrets(box)
	return s, context.Background(), func() { db.Close() }
}

func TestLiveMFAEnrollConfirm(t *testing.T) {
	s, ctx, done := liveStore(t)
	defer done()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	db, _ := database.Open(dbURL)
	defer db.Close()
	org := "11111111-1111-1111-1111-111111111111"
	pw, _ := security.HashPassword("enroll-123")
	_, _ = db.ExecContext(ctx, `INSERT INTO users(id,org_id,username,password_hash) VALUES(gen_random_uuid(),$1,'enroll-user',$2) ON CONFLICT DO NOTHING`, org, pw)
	_, _ = db.ExecContext(ctx, `UPDATE users SET password_hash=$2, totp_enabled=false, totp_secret_enc=NULL WHERE username='enroll-user'`, pw)
	defer db.ExecContext(ctx, `DELETE FROM users WHERE username='enroll-user'`)
	secret, url, err := s.EnrollMFA(ctx, "enroll-user", "TestISP")
	if err != nil || secret == "" || url == "" {
		t.Fatalf("enroll: %v", err)
	}
	code, _ := mfa.CodeAt(secret, time.Now())
	if err := s.ConfirmMFA(ctx, "enroll-user", code); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := s.ConfirmMFA(ctx, "enroll-user", "000000"); err == nil {
		t.Fatal("wrong code must fail")
	}
}
