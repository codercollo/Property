-- migrations/000015_create_schedules_table.up.sql
CREATE TABLE IF NOT EXISTS schedules (
    id bigserial PRIMARY KEY,
    property_id bigint NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scheduled_at timestamp(0) with time zone NOT NULL,
    duration_minutes integer NOT NULL DEFAULT 60,
    status text NOT NULL DEFAULT 'pending',
    notes text,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1,
    CONSTRAINT schedules_status_check CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed'))
);

CREATE INDEX IF NOT EXISTS schedules_property_id_idx ON schedules(property_id);
CREATE INDEX IF NOT EXISTS schedules_user_id_idx ON schedules(user_id);
CREATE INDEX IF NOT EXISTS schedules_agent_id_idx ON schedules(agent_id);
CREATE INDEX IF NOT EXISTS schedules_scheduled_at_idx ON schedules(scheduled_at);
CREATE INDEX IF NOT EXISTS schedules_status_idx ON schedules(status);