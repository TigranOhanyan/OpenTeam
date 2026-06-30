-- +goose Up
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL UNIQUE,
    llm_response_id TEXT NOT NULL UNIQUE,
    tool_call JSON NOT NULL,
    tool_requirement_message_id TEXT NULL UNIQUE,
    tool_result_message_id TEXT NULL UNIQUE,
    FOREIGN KEY (execution_id) REFERENCES executions(id),
    FOREIGN KEY (tool_requirement_message_id) REFERENCES messages(id),
    FOREIGN KEY (tool_result_message_id) REFERENCES messages(id)
);