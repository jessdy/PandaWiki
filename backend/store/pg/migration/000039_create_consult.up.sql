-- 「疑难咨询」主表：每条 = 一个用户提的咨询单。
-- 不区分 kb_id：同一账号在不同站点登录会产生不同的 auths.id，因此天然以 user_id 隔离用户视图。
CREATE TABLE IF NOT EXISTS consult_inquiries (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    contact TEXT NOT NULL DEFAULT '',
    attachments JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    info JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consult_inquiries_user_last ON consult_inquiries(user_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_consult_inquiries_status_last ON consult_inquiries(status, last_message_at DESC);

COMMENT ON COLUMN consult_inquiries.status IS 'pending: 待处理 / processing: 处理中 / replied: 已回复 / closed: 已关闭';
COMMENT ON COLUMN consult_inquiries.attachments IS '附件 URL 列表 [{type:image|video, url, name, size, mime}]';
COMMENT ON COLUMN consult_inquiries.contact IS '用户填写的联系方式（自由文本）';

-- 线程消息表：一个咨询单下多轮 user/admin 互动消息。
CREATE TABLE IF NOT EXISTS consult_messages (
    id BIGSERIAL PRIMARY KEY,
    inquiry_id BIGINT NOT NULL REFERENCES consult_inquiries(id) ON DELETE CASCADE,
    sender_kind VARCHAR(16) NOT NULL,
    sender_id TEXT NOT NULL DEFAULT '',
    sender_name TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    attachments JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consult_messages_inquiry_created ON consult_messages(inquiry_id, created_at);

COMMENT ON COLUMN consult_messages.sender_kind IS 'user: 提问人追加；admin: 管理员回复';
