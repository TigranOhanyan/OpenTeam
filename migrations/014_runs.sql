-- +goose Up
CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    mention_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'completed')) DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mention_id) REFERENCES mentions(id)
);