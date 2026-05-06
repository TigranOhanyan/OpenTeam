-- name: CreateRole :one
INSERT INTO roles (id, member_name, channel_name) VALUES (?, ?, ?) RETURNING *;