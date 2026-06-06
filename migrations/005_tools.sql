-- +goose Up
CREATE TABLE tools (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    tool JSON NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);