-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? LIMIT 1;