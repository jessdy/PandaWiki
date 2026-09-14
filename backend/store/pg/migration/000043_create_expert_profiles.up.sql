CREATE TABLE IF NOT EXISTS expert_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL DEFAULT '',
    introduction VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE expert_profiles IS '具备咨询管理权限的管理员全局专家资料';
