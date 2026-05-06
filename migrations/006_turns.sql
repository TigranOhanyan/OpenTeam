-- +goose Up
CREATE TABLE turns (
    id TEXT PRIMARY KEY,
    prev_id TEXT,
    kind TEXT NOT NULL CHECK(kind IN ('thinking', 'thought', 'acting', 'acted', 'articulation', 'reply')),
    status TEXT NOT NULL CHECK(status IN ('pending', 'completed')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (prev_id) REFERENCES turns(id)
);