-- +goose Up
CREATE TABLE llm_chunk_responses (
    execution_id TEXT NOT NULL,
    sequence_number INT NOT NULL,
    openai_chunk_response JSON NOT NULL,
    PRIMARY KEY (execution_id, sequence_number),
    FOREIGN KEY (execution_id) REFERENCES llm_responses(execution_id)
);