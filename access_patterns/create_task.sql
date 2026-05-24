-- name: CreateTask :one
INSERT INTO tasks (id, role_id, prev_id, instruction, model, stream_mode) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;