- Drop the trigger on payments table
DROP TRIGGER IF EXISTS trigger_update_payments_updated_at ON payments;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_payments_updated_at();

-- Drop all indexes on payments table
DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_property_id;
DROP INDEX IF EXISTS idx_payments_agent_id;

-- Drop the payments table (CASCADE will drop dependent objects)
DROP TABLE IF EXISTS payments CASCADE;

-- Drop the index on properties.agent_id
DROP INDEX IF EXISTS idx_properties_agent_id;

-- Drop the foreign key constraint on properties table
ALTER TABLE properties 
DROP CONSTRAINT IF EXISTS fk_properties_agent;

-- Remove the agent_id column from properties table
ALTER TABLE properties 
DROP COLUMN IF EXISTS agent_id;