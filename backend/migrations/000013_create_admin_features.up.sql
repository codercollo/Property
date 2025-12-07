-- Migration: Add admin features and agent profiles table
-- This migration adds the agent_profiles table for tracking agent verification and status

-- Create agent_profiles table
CREATE TABLE IF NOT EXISTS agent_profiles (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    verified boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'pending')),
    verification_date timestamp(0) with time zone,
    suspension_date timestamp(0) with time zone,
    notes text,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT agent_profiles_user_id_unique UNIQUE (user_id)
);

-- Create index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_agent_profiles_user_id ON agent_profiles(user_id);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);

-- Create index on verified for filtering
CREATE INDEX IF NOT EXISTS idx_agent_profiles_verified ON agent_profiles(verified);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_agent_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_agent_profiles_updated_at
    BEFORE UPDATE ON agent_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_agent_profiles_updated_at();

-- Insert admin permissions if they don't exist (no ON CONFLICT needed since we check first)
DO $$
BEGIN
    -- Insert only if permission doesn't exist
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:users:read') THEN
        INSERT INTO permissions (code) VALUES ('admin:users:read');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:users:write') THEN
        INSERT INTO permissions (code) VALUES ('admin:users:write');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:users:delete') THEN
        INSERT INTO permissions (code) VALUES ('admin:users:delete');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:agents:read') THEN
        INSERT INTO permissions (code) VALUES ('admin:agents:read');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:agents:verify') THEN
        INSERT INTO permissions (code) VALUES ('admin:agents:verify');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:agents:suspend') THEN
        INSERT INTO permissions (code) VALUES ('admin:agents:suspend');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:properties:read') THEN
        INSERT INTO permissions (code) VALUES ('admin:properties:read');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:properties:delete') THEN
        INSERT INTO permissions (code) VALUES ('admin:properties:delete');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'admin:stats:read') THEN
        INSERT INTO permissions (code) VALUES ('admin:stats:read');
    END IF;
END $$;

-- Create a function to grant admin permissions
CREATE OR REPLACE FUNCTION grant_admin_permissions(admin_user_id bigint)
RETURNS void AS $$
BEGIN
    -- Grant all admin permissions
    INSERT INTO users_permissions (user_id, permission_id)
    SELECT admin_user_id, id 
    FROM permissions 
    WHERE code LIKE 'admin:%'
    ON CONFLICT (user_id, permission_id) DO NOTHING;
    
    -- Grant all standard permissions
    INSERT INTO users_permissions (user_id, permission_id)
    SELECT admin_user_id, id 
    FROM permissions 
    WHERE code NOT LIKE 'admin:%'
    ON CONFLICT (user_id, permission_id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Create indexes for performance optimization on existing tables
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_activated ON users(activated);
CREATE INDEX IF NOT EXISTS idx_properties_agent_id ON properties(agent_id);
CREATE INDEX IF NOT EXISTS idx_properties_featured_at ON properties(featured_at);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status);
CREATE INDEX IF NOT EXISTS idx_payments_agent_id ON payments(agent_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);

-- Add comments for documentation
COMMENT ON TABLE agent_profiles IS 'Stores additional profile information and status for agent users';
COMMENT ON COLUMN agent_profiles.verified IS 'Whether the agent has been verified by an admin';
COMMENT ON COLUMN agent_profiles.status IS 'Current status: active, suspended, or pending';
COMMENT ON COLUMN agent_profiles.verification_date IS 'Timestamp when the agent was verified';
COMMENT ON COLUMN agent_profiles.suspension_date IS 'Timestamp when the agent was suspended (if applicable)';
COMMENT ON COLUMN agent_profiles.notes IS 'Admin notes about the agent';