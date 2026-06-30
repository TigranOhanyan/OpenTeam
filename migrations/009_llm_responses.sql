-- +goose Up
CREATE TABLE llm_responses (
    execution_id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind TEXT NOT NULL CHECK (kind IN ('bulk', 'chunk')),
    FOREIGN KEY (execution_id) REFERENCES executions(id)
);