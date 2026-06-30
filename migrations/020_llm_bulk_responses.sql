-- +goose Up
CREATE TABLE llm_bulk_responses (
    id TEXT PRIMARY KEY,
    openai_response JSON NOT NULL,
    FOREIGN KEY (id) REFERENCES llm_responses(id)
);