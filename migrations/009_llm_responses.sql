-- +goose Up
CREATE TABLE llm_responses (
    id TEXT PRIMARY KEY,
    step_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    openai_response JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (step_id) REFERENCES steps(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);