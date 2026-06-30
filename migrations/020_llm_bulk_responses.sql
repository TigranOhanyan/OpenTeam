-- +goose Up
CREATE TABLE llm_bulk_responses (
    execution_id TEXT PRIMARY KEY,
    openai_response JSON NOT NULL,
    FOREIGN KEY (execution_id) REFERENCES llm_responses(execution_id)
);