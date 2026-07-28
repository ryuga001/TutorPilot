DROP TABLE IF EXISTS lecture_attendance;

ALTER TABLE batch_drive_nodes DROP COLUMN IF EXISTS is_system;

UPDATE lectures SET status = 'ended' WHERE status = 'cancelled';

ALTER TABLE lectures
DROP CONSTRAINT IF EXISTS lectures_recording_status_chk,
    DROP CONSTRAINT IF EXISTS lectures_status_chk,
    DROP COLUMN IF EXISTS actual_start_at,
    DROP COLUMN IF EXISTS recording_size_bytes,
    DROP COLUMN IF EXISTS recording_duration_seconds,
    DROP COLUMN IF EXISTS recording_node_id,
    DROP COLUMN IF EXISTS recording_object_key,
    DROP COLUMN IF EXISTS recording_status;


DELETE FROM email_templates
WHERE customer_id = 1 AND name IN ('member_invite', 'member_credentials_reset');

