-- Drop the full-text search index on title if it exists
DROP INDEX IF EXISTS properties_title_idx;

-- Drop the GIN index on the features array if it exists
DROP INDEX IF EXISTS properties_features_idx;
