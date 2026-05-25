-- +goose Up
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    step_id TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    name TEXT NOT NULL,
    arguments JSON NOT NULL,
    FOREIGN KEY (step_id) REFERENCES steps(id)
);