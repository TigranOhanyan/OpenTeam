-- +goose Up
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE,
    step_id TEXT NOT NULL,
    tool_call JSON NOT NULL,
    tool_requirement_message_id TEXT NULL UNIQUE,
    tool_result_message_id TEXT NULL UNIQUE,
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (step_id) REFERENCES steps(id),
    FOREIGN KEY (tool_requirement_message_id) REFERENCES messages(id),
    FOREIGN KEY (tool_result_message_id) REFERENCES messages(id)
);