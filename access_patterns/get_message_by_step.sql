-- name: GetMessageByStep :many
SELECT * FROM messages WHERE step_id = ? ORDER BY id ASC;