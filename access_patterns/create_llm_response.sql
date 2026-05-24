-- name: CreateLlmResponse :one
INSERT INTO llm_responses (id, step_id, task_id, openai_response) VALUES (?, ?, ?, ?) RETURNING *;