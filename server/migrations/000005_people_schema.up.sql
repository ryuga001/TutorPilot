CREATE TABLE addresses (
    id            INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id   INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    local_address TEXT NOT NULL DEFAULT '',
    city          VARCHAR(120) NOT NULL DEFAULT '',
    state         VARCHAR(120) NOT NULL DEFAULT '',
    country       VARCHAR(120) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tutors (
    id                INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id       INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    first_name        VARCHAR(100) NOT NULL,
    last_name         VARCHAR(100) NOT NULL,
    email             VARCHAR(150) NOT NULL,
    phone_no          VARCHAR(30) NOT NULL DEFAULT '',
    designation       VARCHAR(150) NOT NULL DEFAULT '',
    profile_image_key TEXT,
    address_id        INT REFERENCES addresses(id) ON DELETE SET NULL,
    created_by        INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, email)
);

CREATE TABLE students (
    id                INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id       INT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    first_name        VARCHAR(100) NOT NULL,
    last_name         VARCHAR(100) NOT NULL,
    email             VARCHAR(150) NOT NULL,
    phone_no          VARCHAR(30) NOT NULL DEFAULT '',
    profile_image_key TEXT,
    address_id        INT REFERENCES addresses(id) ON DELETE SET NULL,
    created_by        INT REFERENCES dashboard_users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, email)
);