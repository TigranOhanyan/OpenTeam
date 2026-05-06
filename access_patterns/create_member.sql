-- name: CreateMember :one
INSERT INTO members (name, kind) VALUES (?, ?) RETURNING *;