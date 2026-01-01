CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    size BIGINT NOT NULL,
    data BLOB,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME NOT NULL,
    rating REAL,
    metadata JSON
);
