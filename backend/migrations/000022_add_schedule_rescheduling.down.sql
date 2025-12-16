-- Remove constraint
ALTER TABLE schedules DROP CONSTRAINT IF EXISTS chk_reschedule_limit;

-- Remove index
DROP INDEX IF EXISTS idx_schedules_reschedule_count;

-- Remove columns
ALTER TABLE schedules DROP COLUMN IF EXISTS last_rescheduled_at;
ALTER TABLE schedules DROP COLUMN IF EXISTS original_scheduled_at;
ALTER TABLE schedules DROP COLUMN IF EXISTS reschedule_count;