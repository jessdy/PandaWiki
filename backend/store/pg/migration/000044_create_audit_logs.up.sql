CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    account TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT '',
    event TEXT NOT NULL DEFAULT '',
    object_type TEXT NOT NULL DEFAULT '',
    object_id TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    method TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL DEFAULT '',
    status_code INT NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    prev_hash TEXT NOT NULL,
    hash TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_category_action ON audit_logs (category, action);

COMMENT ON TABLE audit_logs IS '安全审计日志：只允许追加，hash 为 HMAC-SM3(prev_hash, 记录内容) 形成的哈希链';

CREATE OR REPLACE FUNCTION audit_logs_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_logs_no_modify ON audit_logs;
CREATE TRIGGER audit_logs_no_modify
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE PROCEDURE audit_logs_append_only();
