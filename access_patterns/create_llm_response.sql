-- name: CreateLlmResponse :one
INSERT INTO llm_responses (id, turn_id, duty_id, openai_response) VALUES (?, ?, ?, ?) RETURNING *;