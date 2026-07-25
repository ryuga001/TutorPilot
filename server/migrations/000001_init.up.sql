CREATE TABLE IF NOT EXISTS users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    email          TEXT        NOT NULL UNIQUE,
    password_hash  TEXT        NOT NULL,
    salt           TEXT        NOT NULL DEFAULT gen_random_uuid()::text,
    email_verified BOOLEAN     NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
