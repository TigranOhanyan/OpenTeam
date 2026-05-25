-- name: CreateLlmChunkResponses :one
INSERT INTO llm_chunk_responses (id, sequence_number, step_id, task_id, openai_chunk_response) VALUES (?, ?, ?, ?, ?) RETURNING *;