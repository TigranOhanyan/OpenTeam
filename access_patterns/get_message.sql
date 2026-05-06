-- name: GetMessage :one
SELECT * FROM messages WHERE id = ? LIMIT 1;