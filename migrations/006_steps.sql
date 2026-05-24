-- +goose Up
CREATE TABLE steps (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('thinking', 'planning', 'acting', 'acted', 'mention', 'replying')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);