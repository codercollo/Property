-- Migration down: Remove admin features

-- Drop the function for granting admin permissions
DROP FUNCTION IF EXISTS grant_admin_permissions(bigint);

-- Drop indexes created for performance
DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_agent_id;
DROP INDEX IF EXISTS idx_reviews_status;
DROP INDEX IF EXISTS idx_properties_featured_at;
DROP INDEX IF EXISTS idx_properties_agent_id;
DROP INDEX IF EXISTS idx_users_activated;
DROP INDEX IF EXISTS idx_users_role;

-- Remove admin permissions
DELETE FROM users_permissions 
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code LIKE 'admin:%'
);

DELETE FROM permissions WHERE code LIKE 'admin:%';

-- Drop trigger and function for agent_profiles
DROP TRIGGER IF EXISTS trigger_update_agent_profiles_updated_at ON agent_profiles;
DROP FUNCTION IF EXISTS update_agent_profiles_updated_at();

-- Drop indexes on agent_profiles
DROP INDEX IF EXISTS idx_agent_profiles_verified;
DROP INDEX IF EXISTS idx_agent_profiles_status;
DROP INDEX IF EXISTS idx_agent_profiles_user_id;

-- Drop agent_profiles table
DROP TABLE IF EXISTS agent_profiles;