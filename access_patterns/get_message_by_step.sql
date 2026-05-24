-- name: GetMessageByStep :one
SELECT * FROM messages WHERE step_id = ? LIMIT 1;