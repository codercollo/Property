-- Add profile_photo column to users table
ALTER TABLE users ADD COLUMN profile_photo TEXT;

-- Create index for faster lookups
CREATE INDEX idx_users_profile_photo ON users(id) WHERE profile_photo IS NOT NULL;