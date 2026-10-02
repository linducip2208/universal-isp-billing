-- 006_password_policy.sql : rotation + login tracking
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
-- 90-day rotation is enforced in store.Authenticate (rotate flag in login response).
