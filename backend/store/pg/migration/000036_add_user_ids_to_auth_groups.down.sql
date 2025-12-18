-- Remove user_ids column from auth_groups table
ALTER TABLE auth_groups DROP COLUMN IF EXISTS user_ids;

