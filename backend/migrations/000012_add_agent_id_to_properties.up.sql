-- Add agent_id column to properties table
ALTER TABLE properties 
ADD COLUMN agent_id BIGINT;

-- Add foreign key constraint to users table
ALTER TABLE properties
ADD CONSTRAINT fk_properties_agent
FOREIGN KEY (agent_id) REFERENCES users(id) ON DELETE CASCADE;

-- Create index for faster lookups by agent
CREATE INDEX idx_properties_agent_id ON properties(agent_id);