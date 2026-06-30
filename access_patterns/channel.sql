-- name: CreateChannel :one
INSERT INTO channels (name, description) VALUES (?, ?) RETURNING *;

-- name: GetChannel :one
SELECT * FROM channels WHERE name = ? LIMIT 1;

-- name: GetChannels :many
SELECT * FROM channels;