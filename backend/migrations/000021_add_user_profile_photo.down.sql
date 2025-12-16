-- Remove index
DROP INDEX IF EXISTS idx_users_profile_photo;

-- Remove profile_photo column
ALTER TABLE users DROP COLUMN IF EXISTS profile_photo;