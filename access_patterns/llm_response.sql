-- name: CreateLlmResponse :one
INSERT INTO llm_responses (execution_id, kind) VALUES (?, ?) RETURNING *;

-- name: GetLlmResponse :one
SELECT * FROM llm_responses WHERE execution_id = ? LIMIT 1;