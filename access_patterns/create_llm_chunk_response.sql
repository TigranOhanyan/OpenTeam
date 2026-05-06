-- name: CreateLlmChunkResponses :one
INSERT INTO llm_chunk_responses (id, sequence_number, turn_id, duty_id, openai_chunk_response) VALUES (?, ?, ?, ?, ?) RETURNING *;