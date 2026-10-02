-- 003_rules.sql : default automation rules (IF overdue+grace expired THEN suspend, etc.)
INSERT INTO automation_rules (org_id, name, conditions, action, params, enabled) VALUES
('11111111-1111-1111-1111-111111111111', 'Suspend on overdue + grace expired',
 '[{"field":"invoice.status","op":"eq","value":"overdue"},{"field":"grace.expired","op":"eq","value":true}]',
 'suspend_subscriber', '{}', true),
('11111111-1111-1111-1111-111111111111', 'Activate on payment confirmed',
 '[{"field":"payment.status","op":"eq","value":"paid"}]',
 'activate_subscriber', '{}', true),
('11111111-1111-1111-1111-111111111111', 'Alert on device offline',
 '[{"field":"device.status","op":"eq","value":"offline"}]',
 'create_alert', '{"severity":"critical"}', true),
('11111111-1111-1111-1111-111111111111', 'Notify NOC when AP offline > 5 min',
 '[{"field":"ap.offline_minutes","op":"gt","value":5}]',
 'notify_noc', '{}', true),
('11111111-1111-1111-1111-111111111111', 'Alert on high CPU',
 '[{"field":"cpu","op":"gt","value":85}]',
 'create_alert', '{"severity":"warning"}', true);
