-- +goose Up
CREATE TABLE llm_chunk_responses (
    id TEXT NOT NULL,
    sequence_number INT NOT NULL,
    turn_id TEXT NOT NULL,
    duty_id TEXT NOT NULL,
    openai_chunk_response JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, sequence_number),
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (duty_id) REFERENCES duties(id)
);