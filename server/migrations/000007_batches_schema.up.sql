CREATE TABLE batches (
    id           INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id  INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    course_id    INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    name         VARCHAR(200) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_by   INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE batch_module_tutors (
    id                INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id          INT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    course_module_id  INT NOT NULL REFERENCES course_modules(id) ON DELETE CASCADE,
    tutor_id          INT NOT NULL REFERENCES tutors(id) ON DELETE CASCADE,
    start_date        DATE,
    expected_end_date DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_id, course_module_id)
);

CREATE TABLE batch_tutors (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id   INT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    tutor_id   INT NOT NULL REFERENCES tutors(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_id, tutor_id)
);

CREATE TABLE batch_students (
    id           INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id     INT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    student_id   INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    enrolled_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_id, student_id)
);

CREATE TABLE batch_drive_nodes (
    id           INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id     INT NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    parent_id    INT REFERENCES batch_drive_nodes(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    node_type    VARCHAR(10) NOT NULL,
    object_key   TEXT,
    content_type VARCHAR(120) NOT NULL DEFAULT '',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    created_by   INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
