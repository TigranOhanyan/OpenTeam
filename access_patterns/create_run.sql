-- name: CreateRun :one
INSERT INTO runs (id, kind) VALUES (?, ?) RETURNING *;