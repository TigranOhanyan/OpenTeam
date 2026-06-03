-- name: CreateRun :one
INSERT INTO runs (id, mention_id) VALUES (?, ?) RETURNING *;