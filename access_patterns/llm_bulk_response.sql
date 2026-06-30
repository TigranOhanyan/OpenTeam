-- name: CreateLlmBulkResponse :one
INSERT INTO llm_bulk_responses (execution_id, openai_response) VALUES (?, ?) RETURNING *;

-- name: GetLlmBulkResponse :one
SELECT * FROM llm_bulk_responses WHERE execution_id = ? LIMIT 1;