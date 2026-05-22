-- +goose Up
CREATE TABLE handoffs (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    FOREIGN KEY (turn_id) REFERENCES turns(id)
);