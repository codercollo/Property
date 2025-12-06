CREATE TABLE IF NOT EXISTS reviews (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    property_id bigint NOT NULL REFERENCES properties ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    rating integer NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    approved_at timestamp(0) with time zone,
    approved_by bigint REFERENCES users ON DELETE SET NULL,
    version integer NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS reviews_property_id_idx ON reviews(property_id);
CREATE INDEX IF NOT EXISTS reviews_user_id_idx ON reviews(user_id);
CREATE INDEX IF NOT EXISTS reviews_status_idx ON reviews(status);
CREATE INDEX IF NOT EXISTS reviews_created_at_idx ON reviews(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS reviews_user_property_unique_idx ON reviews(user_id, property_id);

INSERT INTO permissions (code)
SELECT 'reviews:read'
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'reviews:read');

INSERT INTO permissions (code)
SELECT 'reviews:write'
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'reviews:write');

INSERT INTO permissions (code)
SELECT 'reviews:moderate'
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'reviews:moderate');