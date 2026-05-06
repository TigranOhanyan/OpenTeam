-- name: GetMessageByTurn :one
SELECT * FROM messages WHERE turn_id = ? LIMIT 1;