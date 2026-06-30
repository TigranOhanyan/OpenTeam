-- name: CreateLlmChunkResponses :exec
INSERT INTO llm_chunk_responses (execution_id, sequence_number, openai_chunk_response) VALUES (?, ?, ?);

-- name: GetLlmChunkResponse :many
SELECT * FROM llm_chunk_responses WHERE execution_id = ? ORDER BY sequence_number ASC;
