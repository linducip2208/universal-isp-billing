-- 010_mfa.down.sql
DROP TABLE IF EXISTS totp_backup_codes;
ALTER TABLE users DROP COLUMN IF EXISTS totp_secret_enc;
ALTER TABLE users DROP COLUMN IF EXISTS totp_enabled;
