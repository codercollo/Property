DELETE FROM permissions WHERE code IN ('reviews:moderate', 'reviews:write', 'reviews:read');
DROP INDEX IF EXISTS reviews_user_property_unique_idx;
DROP INDEX IF EXISTS reviews_created_at_idx;
DROP INDEX IF EXISTS reviews_status_idx;
DROP INDEX IF EXISTS reviews_user_id_idx;
DROP INDEX IF EXISTS reviews_property_id_idx;
DROP TABLE IF EXISTS reviews CASCADE;