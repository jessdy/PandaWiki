DROP INDEX IF EXISTS "idx_users_region";
ALTER TABLE "public"."users" DROP COLUMN IF EXISTS "region";
