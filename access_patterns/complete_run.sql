-- name: CompleteRun :one
UPDATE runs SET status = 'completed' WHERE id = ? RETURNING *;