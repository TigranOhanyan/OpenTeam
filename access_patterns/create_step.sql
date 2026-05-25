-- name: CreateStep :one
INSERT INTO steps (id, run_id, kind) VALUES (?, ?, ?) RETURNING *;