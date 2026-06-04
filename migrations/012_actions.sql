-- +goose Up
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE,
    step_id TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    tool_call JSON NOT NULL,
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (step_id) REFERENCES steps(id)
);