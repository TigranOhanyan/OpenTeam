-- name: GetMessagesByDuty :many
SELECT * FROM messages WHERE duty_id = ? ORDER BY created_at ASC;