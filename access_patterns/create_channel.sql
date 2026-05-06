-- name: CreateChannel :one
INSERT INTO channels (name, description) VALUES (?, ?) RETURNING *;