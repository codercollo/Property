CREATE TABLE IF NOT EXISTS user_favourites (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    property_id bigint NOT NULL REFERENCES properties ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    
    -- Composite primary key ensures each user can favourite a property only once
    PRIMARY KEY (user_id, property_id)
);

-- Indexes for efficient queries
CREATE INDEX idx_user_favourites_user_id ON user_favourites(user_id);
CREATE INDEX idx_user_favourites_property_id ON user_favourites(property_id);
CREATE INDEX idx_user_favourites_created_at ON user_favourites(created_at DESC);

-- Composite index for user's favourites ordered by date
CREATE INDEX idx_user_favourites_user_created 
    ON user_favourites(user_id, created_at DESC);

-- Add a comment to the table
COMMENT ON TABLE user_favourites IS 'Stores user saved/favourite properties for quick access';
