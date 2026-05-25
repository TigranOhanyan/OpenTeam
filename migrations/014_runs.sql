-- +goose Up
CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    source_step_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_step_id) REFERENCES steps(id)
);