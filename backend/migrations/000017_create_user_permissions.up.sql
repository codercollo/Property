BEGIN;

-- 1. Create user_permissions table
CREATE TABLE IF NOT EXISTS user_permissions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY(user_id, permission)
);

-- 2. Seed initial permissions
-- Optional: Adjust user_id for your admin user
INSERT INTO user_permissions (user_id, permission)
VALUES
    (25, 'properties:read'),
    (25, 'properties:write'),
    (25, 'properties:delete'),
    (25, 'properties:feature'),
    (25, 'agents:manage'),
    (25, 'reviews:read'),
    (25, 'reviews:write'),
    (25, 'reviews:moderate'),
    (25, 'users:manage'),
    (25, 'inquiries:create'),
    (25, 'inquiries:read')
ON CONFLICT DO NOTHING;

COMMIT;