-- +goose Up
CREATE TABLE llm_responses (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    duty_id TEXT NOT NULL,
    openai_response JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (duty_id) REFERENCES duties(id)
);