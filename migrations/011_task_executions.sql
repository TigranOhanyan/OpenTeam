-- +goose Up
CREATE TABLE task_executions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    FOREIGN KEY (id) REFERENCES tasks(id),
    FOREIGN KEY (execution_id) REFERENCES executions(id)
);