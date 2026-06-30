-- +goose Up
CREATE TABLE llm_chunk_responses (
    id TEXT NOT NULL,
    sequence_number INT NOT NULL,
    openai_chunk_response JSON NOT NULL,
    PRIMARY KEY (id, sequence_number),
    FOREIGN KEY (id) REFERENCES llm_responses(id)
);