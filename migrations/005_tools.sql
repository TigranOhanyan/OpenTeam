-- +goose Up
CREATE TABLE tools (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    parameters JSON NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);