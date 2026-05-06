-- name: GetDuty :one
SELECT * FROM duties WHERE id = ? LIMIT 1;