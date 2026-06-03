-- name: CompleteStep :one
UPDATE steps SET status = 'completed' WHERE id = ? RETURNING *;