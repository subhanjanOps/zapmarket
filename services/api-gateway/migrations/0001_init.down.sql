DROP TABLE IF EXISTS gateway_audit_log;
DROP TRIGGER IF EXISTS gateway_routes_changed ON gateway_routes;
DROP FUNCTION IF EXISTS gateway_routes_notify();
DROP TRIGGER IF EXISTS gateway_routes_updated_at ON gateway_routes;
DROP FUNCTION IF EXISTS gateway_routes_set_updated_at();
DROP TABLE IF EXISTS gateway_routes;
