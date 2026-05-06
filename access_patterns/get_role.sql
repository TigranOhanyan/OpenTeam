-- name: GetRole :one
SELECT * FROM roles WHERE id = ? LIMIT 1;