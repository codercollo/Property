
-- First, add the new columns
ALTER TABLE agent_profiles 
ADD COLUMN IF NOT EXISTS rejection_reason TEXT,
ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMP WITH TIME ZONE;

-- Update the check constraint to include 'rejected' status
-- Drop the existing constraint first
ALTER TABLE agent_profiles 
DROP CONSTRAINT IF EXISTS agent_profiles_status_check;

-- Add the updated constraint with 'rejected' included
ALTER TABLE agent_profiles 
ADD CONSTRAINT agent_profiles_status_check 
CHECK (status IN ('active', 'suspended', 'rejected'));

-- Create an index for querying rejected agents
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status_rejected 
ON agent_profiles(status) 
WHERE status = 'rejected';

-- Add comments explaining the rejection fields
COMMENT ON COLUMN agent_profiles.rejection_reason IS 'Reason provided by admin when rejecting agent verification';
COMMENT ON COLUMN agent_profiles.rejected_at IS 'Timestamp when the agent verification was rejected';