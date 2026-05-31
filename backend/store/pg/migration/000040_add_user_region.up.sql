ALTER TABLE "public"."users" ADD COLUMN IF NOT EXISTS "region" text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS "idx_users_region" ON "public"."users" ("region") WHERE "region" <> '';
