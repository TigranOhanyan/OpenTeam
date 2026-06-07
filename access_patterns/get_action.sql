-- name: GetAction :one
SELECT * FROM actions WHERE id = ? LIMIT 1;