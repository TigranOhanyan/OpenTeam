-- name: GetChannel :one
SELECT * FROM channels WHERE name = ? LIMIT 1;