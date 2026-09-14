ALTER TABLE conversation_messages
ADD COLUMN IF NOT EXISTS retrieval_content TEXT NOT NULL DEFAULT '';
