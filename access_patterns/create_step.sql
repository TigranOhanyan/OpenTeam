-- name: CreateStep :one
INSERT INTO steps (id, run_id, task_id) VALUES (?, ?, ?) RETURNING *;