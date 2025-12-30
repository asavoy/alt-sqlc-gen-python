CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT,
    age        INTEGER,
    created_at TIMESTAMPTZ NOT NULL
);
