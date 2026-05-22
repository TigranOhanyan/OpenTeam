-- name: CreateTurn :one
INSERT INTO turns (id, kind) VALUES (?, ?) RETURNING *;