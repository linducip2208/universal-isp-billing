-- 008_demo_flag.sql : explicit demo marking (Phase 40). Demo data must NEVER
-- be confused with real device data; production paths filter is_demo=false.
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT false;
UPDATE organizations SET is_demo = true WHERE id = '11111111-1111-1111-1111-111111111111';
