-- Add reschedule tracking columns to schedules table
ALTER TABLE schedules ADD COLUMN reschedule_count INTEGER DEFAULT 0;
ALTER TABLE schedules ADD COLUMN original_scheduled_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE schedules ADD COLUMN last_rescheduled_at TIMESTAMP WITH TIME ZONE;

-- Create index for tracking rescheduled schedules
CREATE INDEX idx_schedules_reschedule_count ON schedules(reschedule_count) WHERE reschedule_count > 0;

-- Add constraint to limit excessive rescheduling (max 3 reschedules)
ALTER TABLE schedules ADD CONSTRAINT chk_reschedule_limit CHECK (reschedule_count <= 3);