-- name: GetStep :one
SELECT * FROM steps WHERE id = ? LIMIT 1;