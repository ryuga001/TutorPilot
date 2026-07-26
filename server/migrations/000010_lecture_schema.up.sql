CREATE TABLE IF NOT EXISTS lectures (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id INT NOT NULL
    REFERENCES customers(id) ON DELETE CASCADE,
    batch_id INT NOT NULL
    REFERENCES batches(id) ON DELETE CASCADE,
    module_id INT
    REFERENCES course_modules(id) ON DELETE SET NULL,
    tutor_id INT
    REFERENCES tutors(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    room_name VARCHAR(255) UNIQUE,
    egress_id TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    recording_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    recording_url TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    created_by INT
    REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO privileges (name, type) VALUES
('lecture.create', 'lecture'),
('lecture.view',   'lecture'),
('lecture.edit',   'lecture'),
('lecture.delete', 'lecture'),
('lecture.control', 'lecture');

INSERT INTO role_privileges (role_id, privilege_id)
SELECT r.id, p.id
FROM roles r
JOIN privileges p ON p.name = ANY (
    ARRAY[
        'lecture.create',
        'lecture.view',
        'lecture.edit',
        'lecture.delete',
        'lecture.control'
    ]
)
WHERE r.name = 'Admin'
    ON CONFLICT DO NOTHING;