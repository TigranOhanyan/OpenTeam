-- name: CreateLlmRequest :one
INSERT INTO llm_requests (execution_id, task_id, openai_request) VALUES (?, ?, ?) RETURNING *;


-- name: GetLlmRequest :one
SELECT * FROM llm_requests WHERE execution_id = ? LIMIT 1;
