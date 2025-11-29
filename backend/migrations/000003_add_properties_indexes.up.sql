-- GIN index for full-text search on the title column
CREATE INDEX IF NOT EXISTS properties_title_idx 
ON properties USING GIN (to_tsvector('simple', title));

-- GIN index for the features array column
CREATE INDEX IF NOT EXISTS properties_features_idx 
ON properties USING GIN (features);
