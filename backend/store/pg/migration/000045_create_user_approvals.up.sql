CREATE TABLE IF NOT EXISTS user_approvals (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    summary TEXT NOT NULL DEFAULT '',
    target_user_id TEXT NOT NULL DEFAULT '',
    target_account TEXT NOT NULL DEFAULT '',
    kb_id TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    requester_id TEXT NOT NULL,
    requester_account TEXT NOT NULL DEFAULT '',
    reviewer_id TEXT NOT NULL DEFAULT '',
    reviewer_account TEXT NOT NULL DEFAULT '',
    review_comment TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_approvals_status ON user_approvals (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_approvals_requester ON user_approvals (requester_id);

COMMENT ON TABLE user_approvals IS '三员分立：系统管理员发起的用户增删与权限变更，经安全管理员审批后生效';
