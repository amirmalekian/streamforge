-- Migration: 000004_remove_updated_at_triggers.down.sql
DROP TRIGGER IF EXISTS update_media_items_updated_at ON media_items;
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Migration: 000003_drop_media_items_table.down.sql
DROP INDEX IF EXISTS idx_media_items_status;
DROP INDEX IF EXISTS idx_media_items_job_id;
DROP TABLE IF EXISTS media_items;
DROP TYPE IF EXISTS media_item_status;

-- Migration: 000002_drop_jobs_table.down.sql
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_user_id;
DROP TABLE IF EXISTS jobs;
DROP TYPE IF EXISTS job_status;

-- Migration: 000001_drop_users_table.down.sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;