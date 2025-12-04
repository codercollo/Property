-- Remove featured_at column if rolling back
ALTER TABLE properties
DROP COLUMN IF EXISTS featured_at;
