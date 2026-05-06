-- name: CreateDuty :one
INSERT INTO duties (id, role_id, prev_id, instruction, model, stream_mode) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;