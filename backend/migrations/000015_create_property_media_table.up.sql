-- migrations/000007_create_property_media_table.up.sql

CREATE TABLE IF NOT EXISTS property_media (
    id bigserial PRIMARY KEY,
    property_id bigint NOT NULL REFERENCES properties ON DELETE CASCADE,
    media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('image', 'video', 'floor_plan')),
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_size bigint NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    display_order integer NOT NULL DEFAULT 0,
    caption TEXT,
    is_primary boolean NOT NULL DEFAULT false,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1
);

-- Create indexes for efficient queries
CREATE INDEX idx_property_media_property_id ON property_media(property_id);
CREATE INDEX idx_property_media_type ON property_media(media_type);
CREATE INDEX idx_property_media_display_order ON property_media(property_id, display_order);

-- Ensure only one primary media per property per type
CREATE UNIQUE INDEX idx_property_media_primary ON property_media(property_id, media_type, is_primary) 
WHERE is_primary = true;