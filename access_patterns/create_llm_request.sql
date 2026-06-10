-- name: CreateLlmRequest :one
INSERT INTO llm_requests (id, step_id, task_id, openai_request) VALUES (?, ?, ?, ?) RETURNING *;