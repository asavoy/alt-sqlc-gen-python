CREATE TABLE authors (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    born_on    DATE NOT NULL,
    updated_at TIMESTAMP
);

CREATE TABLE books (
    id           BIGSERIAL PRIMARY KEY,
    author_id    BIGINT NOT NULL REFERENCES authors(id),
    title        TEXT NOT NULL,
    published_at TIMESTAMP NOT NULL
);
