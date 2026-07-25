CREATE TABLE courses (
    id             INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id    INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    title          VARCHAR(200) NOT NULL,
    slug           VARCHAR(220) NOT NULL,
    summary        TEXT NOT NULL DEFAULT '',
    description_md TEXT NOT NULL DEFAULT '',
    thumbnail_key  TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at   TIMESTAMPTZ,
    created_by     INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, slug)
);

CREATE TABLE course_modules (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id  INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title      VARCHAR(200) NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE course_lessons (
    id         INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    module_id  INT NOT NULL REFERENCES course_modules(id) ON DELETE CASCADE,
    title      VARCHAR(200) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE course_resources (
    id           INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id    INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    lesson_id    INT REFERENCES course_lessons(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    object_key   TEXT NOT NULL,
    content_type VARCHAR(120) NOT NULL DEFAULT '',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    created_by   INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
