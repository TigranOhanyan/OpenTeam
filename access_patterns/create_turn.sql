-- name: CreateTurn :one
INSERT INTO turns (id, prev_id, kind, status) VALUES (?, ?, ?, ?) RETURNING *;