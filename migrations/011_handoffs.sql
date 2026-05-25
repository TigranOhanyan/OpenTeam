-- +goose Up
CREATE TABLE handoffs (
    id TEXT PRIMARY KEY,
    step_id TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    FOREIGN KEY (step_id) REFERENCES steps(id)
);