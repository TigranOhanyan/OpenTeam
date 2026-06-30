-- +goose Up
CREATE TABLE task_executions (
    execution_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (execution_id) REFERENCES executions(id)
);