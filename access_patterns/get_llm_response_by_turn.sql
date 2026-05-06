-- name: GetLlmResponseByTurn :one
SELECT * FROM llm_responses WHERE turn_id = ? LIMIT 1;