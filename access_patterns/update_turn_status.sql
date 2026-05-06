-- name: UpdateTurnStatus :one
UPDATE turns SET status = ?, completed_at = ? WHERE id = ? RETURNING *;