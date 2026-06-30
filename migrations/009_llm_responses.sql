-- +goose Up
CREATE TABLE llm_responses (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind TEXT NOT NULL CHECK (kind IN ('bulk', 'chunk')),
    FOREIGN KEY (execution_id) REFERENCES executions(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);