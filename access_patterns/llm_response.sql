-- name: CreateLlmResponse :one
INSERT INTO llm_responses (id, execution_id, kind) VALUES (?, ?, ?) RETURNING *;


-- name: GetLlmResponseByExecution :one
SELECT * FROM llm_responses WHERE execution_id = ? LIMIT 1;

-- name: GetLlmResponse :one
SELECT * FROM llm_responses WHERE id = ? LIMIT 1;