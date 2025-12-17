-- Drop the index
DROP INDEX IF EXISTS idx_agent_profiles_status_rejected;

-- Remove the columns
ALTER TABLE agent_profiles 
DROP COLUMN IF EXISTS rejection_reason,
DROP COLUMN IF EXISTS rejected_at;

-- Restore the original check constraint (without 'rejected')
ALTER TABLE agent_profiles 
DROP CONSTRAINT IF EXISTS agent_profiles_status_check;

ALTER TABLE agent_profiles 
ADD CONSTRAINT agent_profiles_status_check 
CHECK (status IN ('active', 'suspended'));