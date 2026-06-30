-- name: CreateLlmRequest :one
INSERT INTO llm_requests (id, execution_id, task_id, openai_request) VALUES (?, ?, ?, ?) RETURNING *;


-- name: GetLlmRequestByExecution :one
SELECT * FROM llm_requests WHERE execution_id = ? LIMIT 1;

-- name: GetPartialLlmRequest :one
SELECT id, execution_id, task_id FROM llm_requests WHERE id = ? LIMIT 1;

-- name: GetPartialLlmRequestByExecution :one
SELECT id, execution_id, task_id FROM llm_requests WHERE execution_id = ? LIMIT 1;

