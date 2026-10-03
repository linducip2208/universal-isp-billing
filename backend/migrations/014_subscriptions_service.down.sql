-- 014_subscriptions_service.down.sql
DROP INDEX IF EXISTS idx_subs_org_username;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS service;
