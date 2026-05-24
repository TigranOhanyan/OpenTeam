-- name: CreateStep :one
INSERT INTO steps (id, kind) VALUES (?, ?) RETURNING *;