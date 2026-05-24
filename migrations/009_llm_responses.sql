-- +goose Up
CREATE TABLE llm_responses (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    openai_response JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);