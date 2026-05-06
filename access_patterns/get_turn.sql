-- name: GetTurn :one
SELECT * FROM turns WHERE id = ? LIMIT 1;