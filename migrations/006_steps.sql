-- +goose Up
CREATE TABLE steps (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('observing', 'reasoning', 'acting', 'mentioning')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);