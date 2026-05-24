-- name: GetLlmResponseByStep :one
SELECT * FROM llm_responses WHERE step_id = ? LIMIT 1;