-- 014_subscriptions_service.sql : service type + username uniqueness
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS service TEXT NOT NULL DEFAULT 'pppoe';
CREATE UNIQUE INDEX IF NOT EXISTS idx_subs_org_username ON subscriptions(org_id, username) WHERE deleted_at IS NULL;
