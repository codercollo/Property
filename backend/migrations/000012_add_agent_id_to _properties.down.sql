-- Add agent_id column only if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='properties' AND column_name='agent_id'
    ) THEN
        ALTER TABLE properties ADD COLUMN agent_id BIGINT;
    END IF;
END $$;

-- Add foreign key constraint only if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name='fk_properties_agent'
    ) THEN
        ALTER TABLE properties
        ADD CONSTRAINT fk_properties_agent
        FOREIGN KEY (agent_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Create index only if it doesn't exist
CREATE INDEX IF NOT EXISTS idx_properties_agent_id ON properties(agent_id);