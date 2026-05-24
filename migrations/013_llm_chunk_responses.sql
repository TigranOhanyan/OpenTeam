-- +goose Up
CREATE TABLE llm_chunk_responses (
    id TEXT NOT NULL,
    sequence_number INT NOT NULL,
    turn_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    openai_chunk_response JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, sequence_number),
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);