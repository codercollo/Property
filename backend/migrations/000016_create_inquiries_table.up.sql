
CREATE TABLE IF NOT EXISTS inquiries (
    id bigserial PRIMARY KEY,
    property_id bigint NOT NULL REFERENCES properties ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    agent_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    
    -- Inquiry details
    name varchar(255) NOT NULL,
    email varchar(255) NOT NULL,
    phone varchar(50),
    message text NOT NULL,
    
    -- Inquiry type and scheduling
    inquiry_type varchar(50) NOT NULL DEFAULT 'general',
    preferred_contact_method varchar(50) DEFAULT 'email',
    preferred_viewing_date timestamp with time zone,
    
    -- Status tracking
    status varchar(50) NOT NULL DEFAULT 'new',
    priority varchar(20) DEFAULT 'normal',
    
    -- Agent response
    agent_notes text,
    responded_at timestamp with time zone,
    
    -- Metadata
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1,
    
    -- Constraints
    CONSTRAINT inquiries_inquiry_type_check 
        CHECK (inquiry_type IN ('general', 'viewing', 'purchase', 'rent', 'more_info')),
    CONSTRAINT inquiries_status_check 
        CHECK (status IN ('new', 'contacted', 'scheduled', 'closed', 'spam')),
    CONSTRAINT inquiries_priority_check 
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT inquiries_contact_method_check 
        CHECK (preferred_contact_method IN ('email', 'phone', 'any'))
);

-- Indexes for efficient queries
CREATE INDEX idx_inquiries_property_id ON inquiries(property_id);
CREATE INDEX idx_inquiries_user_id ON inquiries(user_id);
CREATE INDEX idx_inquiries_agent_id ON inquiries(agent_id);
CREATE INDEX idx_inquiries_status ON inquiries(status);
CREATE INDEX idx_inquiries_created_at ON inquiries(created_at DESC);
CREATE INDEX idx_inquiries_agent_status ON inquiries(agent_id, status);

-- Composite index for agent dashboard queries
CREATE INDEX idx_inquiries_agent_priority_created 
    ON inquiries(agent_id, priority DESC, created_at DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_inquiries_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_inquiries_updated_at
    BEFORE UPDATE ON inquiries
    FOR EACH ROW
    EXECUTE FUNCTION update_inquiries_updated_at();