-- +goose Up
CREATE TABLE turns (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('thinking', 'planning', 'acting', 'acted', 'addressing', 'replying')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);