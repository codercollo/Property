-- Add featured_at column to properties table
ALTER TABLE properties
ADD COLUMN featured_at timestamp with time zone;
