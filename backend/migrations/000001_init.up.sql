CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT        NOT NULL CHECK (char_length(email) <= 254),
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness is enforced by the database, not only by Go.
CREATE UNIQUE INDEX users_email_lower_key ON users (LOWER(email));

CREATE TABLE tasks (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       TEXT        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
    description TEXT        NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    completed   BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Covers the only list query: WHERE user_id = ? ORDER BY created_at DESC, id DESC.
CREATE INDEX tasks_user_created_idx ON tasks (user_id, created_at DESC, id DESC);
