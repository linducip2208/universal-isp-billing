-- 001_init.down.sql : full schema rollback (destructive — restore from backup instead in production)
DROP TABLE IF EXISTS journal_lines, journal_entries, invoice_sequences CASCADE;
DROP TABLE IF EXISTS traffic_samples, interface_samples, system_settings CASCADE;
DROP TABLE IF EXISTS webhook_deliveries, webhooks, api_keys, notifications CASCADE;
DROP TABLE IF EXISTS automation_executions, automation_rules CASCADE;
DROP TABLE IF EXISTS network_job_attempts, network_jobs CASCADE;
DROP TABLE IF EXISTS onus, olt_devices CASCADE;
DROP TABLE IF EXISTS radius_accounting, radius_sessions, radius_users CASCADE;
DROP TABLE IF EXISTS payment_transactions, payments, invoice_items, invoices CASCADE;
DROP TABLE IF EXISTS network_services, subscriptions, packages, bandwidth_profiles CASCADE;
DROP TABLE IF EXISTS service_addresses, customer_contacts, customers CASCADE;
DROP TABLE IF EXISTS device_credentials, device_capabilities, devices, connectors CASCADE;
DROP TABLE IF EXISTS ip_addresses, ip_pools, vlans CASCADE;
DROP TABLE IF EXISTS alerts, events, audit_logs CASCADE;
DROP TABLE IF EXISTS sites, user_roles, permissions, roles, users, organizations CASCADE;
