-- name: GetRun :one
SELECT * FROM runs WHERE id = ? LIMIT 1;