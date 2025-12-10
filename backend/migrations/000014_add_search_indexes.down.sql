-- Remove search optimization indexes

DROP INDEX IF EXISTS idx_properties_agent_id;
DROP INDEX IF EXISTS idx_properties_created_at;
DROP INDEX IF EXISTS idx_properties_price_bedrooms;
DROP INDEX IF EXISTS idx_properties_location_type;
DROP INDEX IF EXISTS idx_properties_features_gin;
DROP INDEX IF EXISTS idx_properties_featured_at;
DROP INDEX IF EXISTS idx_properties_area;
DROP INDEX IF EXISTS idx_properties_bathrooms;
DROP INDEX IF EXISTS idx_properties_bedrooms;
DROP INDEX IF EXISTS idx_properties_price;
DROP INDEX IF EXISTS idx_properties_type_lower;
DROP INDEX IF EXISTS idx_properties_location_lower;