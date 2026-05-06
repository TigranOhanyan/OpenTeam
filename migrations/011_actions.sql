-- +goose Up
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    name TEXT NOT NULL,
    arguments JSON NOT NULL,
    FOREIGN KEY (turn_id) REFERENCES turns(id)
);