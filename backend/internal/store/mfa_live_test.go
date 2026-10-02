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

// TestLiveMFA exercises enrollment-shaped rows: password auth flags
// mfaRequired, and a live TOTP code verifies through the encrypted secret.
func TestLiveMFA(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	org := "11111111-1111-1111-1111-111111111111"
	pwHash, _ := security.HashPassword("testpass-123")
	key, _ := security.DeriveKey("test-vault-passphrase-for-mfa-live")
	box, _ := security.NewSecretsBox(key)
	secret, _ := mfa.GenerateSecret()
	enc, _ := box.Encrypt(secret)
	_, err = db.ExecContext(ctx, `INSERT INTO users(id,org_id,username,password_hash,totp_secret_enc,totp_enabled) VALUES(gen_random_uuid(),$1,'mfa-user',$2,$3,true) ON CONFLICT DO NOTHING`, org, pwHash, enc)
	// tolerate reruns: ensure row state
	_, _ = db.ExecContext(ctx, `UPDATE users SET password_hash=$2, totp_secret_enc=$3, totp_enabled=true WHERE username='mfa-user'`, pwHash, enc)
	defer db.ExecContext(ctx, `DELETE FROM totp_backup_codes WHERE user_id=(SELECT id FROM users WHERE username='mfa-user')`)
	defer db.ExecContext(ctx, `DELETE FROM users WHERE username='mfa-user'`)
	if err != nil {
		t.Fatal(err)
	}
	s := store.New(db).WithSecrets(box)
	_, _, _, _, mfaReq, err := s.Authenticate(ctx, "mfa-user", "testpass-123")
	if err != nil {
		t.Fatal(err)
	}
	if !mfaReq {
		t.Fatal("enrolled user must flag mfaRequired")
	}
	code, _ := mfa.CodeAt(secret, time.Now())
	if err := s.VerifyTOTP(ctx, "mfa-user", code); err != nil {
		t.Fatalf("live totp: %v", err)
	}
	if err := s.VerifyTOTP(ctx, "mfa-user", "000000"); err == nil {
		t.Fatal("wrong code must fail")
	}
}
