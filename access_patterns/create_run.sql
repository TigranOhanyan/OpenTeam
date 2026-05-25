-- name: CreateRun :one
INSERT INTO runs (id, source_step_id) VALUES (?, ?) RETURNING *;