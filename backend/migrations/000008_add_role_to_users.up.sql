-- Add role column to users table
ALTER TABLE users 
ADD COLUMN role text NOT NULL DEFAULT 'user';

-- Add check constraint to ensure valid roles
ALTER TABLE users
ADD CONSTRAINT users_role_check 
CHECK (role IN ('user', 'agent', 'admin'));

-- Create index on role for faster queries
CREATE INDEX idx_users_role ON users(role);