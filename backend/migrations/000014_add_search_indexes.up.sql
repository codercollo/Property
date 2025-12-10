-- Add indexes to optimize property search queries


CREATE INDEX IF NOT EXISTS idx_properties_location_lower ON properties (LOWER(location));
CREATE INDEX IF NOT EXISTS idx_properties_type_lower ON properties (LOWER(property_type));
CREATE INDEX IF NOT EXISTS idx_properties_price ON properties (price);
CREATE INDEX IF NOT EXISTS idx_properties_bedrooms ON properties (bedrooms);
CREATE INDEX IF NOT EXISTS idx_properties_bathrooms ON properties (bathrooms);
CREATE INDEX IF NOT EXISTS idx_properties_area ON properties (area);
CREATE INDEX IF NOT EXISTS idx_properties_featured_at ON properties (featured_at) WHERE featured_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_properties_features_gin ON properties USING GIN (features);
CREATE INDEX IF NOT EXISTS idx_properties_location_type ON properties (LOWER(location), LOWER(property_type));
CREATE INDEX IF NOT EXISTS idx_properties_price_bedrooms ON properties (price, bedrooms);
CREATE INDEX IF NOT EXISTS idx_properties_created_at ON properties (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_properties_agent_id ON properties (agent_id) WHERE agent_id IS NOT NULL;