-- +goose Up
CREATE TABLE steps (
    id TEXT PRIMARY KEY,
    run_id TEXT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('observing', 'reasoning', 'acting', 'asking')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (run_id) REFERENCES runs(id)
);