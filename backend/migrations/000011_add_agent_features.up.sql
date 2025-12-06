-- Add agent_id column to properties table
ALTER TABLE properties 
ADD COLUMN agent_id bigint;

-- Add foreign key constraint linking properties to users (agents)
ALTER TABLE properties
ADD CONSTRAINT fk_properties_agent
FOREIGN KEY (agent_id) REFERENCES users(id) ON DELETE CASCADE;

-- Create index on agent_id for faster queries
CREATE INDEX idx_properties_agent_id ON properties(agent_id);

-- Add comment to explain the column
COMMENT ON COLUMN properties.agent_id IS 'References the user ID of the agent who owns this property';

-- =====================================================
-- Create payments table
-- =====================================================

CREATE TABLE payments (
    id bigserial PRIMARY KEY,
    agent_id bigint NOT NULL,
    property_id bigint NOT NULL,
    amount decimal(10, 2) NOT NULL,
    payment_method text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1,
    
    -- Foreign key constraints
    CONSTRAINT fk_payments_agent FOREIGN KEY (agent_id) 
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_payments_property FOREIGN KEY (property_id) 
        REFERENCES properties(id) ON DELETE CASCADE,
    
    -- Check constraints
    CONSTRAINT check_amount_positive CHECK (amount > 0),
    CONSTRAINT check_status_valid CHECK (status IN ('pending', 'completed', 'failed'))
);

-- Create indexes for payments table to optimize queries
CREATE INDEX idx_payments_agent_id ON payments(agent_id);
CREATE INDEX idx_payments_property_id ON payments(property_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);

-- Add comments to document the table
COMMENT ON TABLE payments IS 'Stores payment records for featured property listings';
COMMENT ON COLUMN payments.agent_id IS 'The agent making the payment';
COMMENT ON COLUMN payments.property_id IS 'The property being featured';
COMMENT ON COLUMN payments.amount IS 'Payment amount in the system currency';
COMMENT ON COLUMN payments.payment_method IS 'Payment method used (e.g., credit_card, paypal)';
COMMENT ON COLUMN payments.status IS 'Payment status: pending, completed, or failed';

-- Create trigger function to auto-update updated_at

CREATE OR REPLACE FUNCTION update_payments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger on payments table
CREATE TRIGGER trigger_update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_payments_updated_at();

COMMENT ON FUNCTION update_payments_updated_at() IS 'Automatically updates the updated_at timestamp when a payment record is modified';