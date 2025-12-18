-- Add user_ids column to auth_groups table
-- This column stores user IDs (from users table) associated with the auth group
ALTER TABLE auth_groups ADD COLUMN IF NOT EXISTS user_ids TEXT[] DEFAULT '{}';

