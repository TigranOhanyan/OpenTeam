-- name: CreateRole :one
INSERT INTO roles (id, member_name, channel_name) VALUES (?, ?, ?) RETURNING *;

-- name: GetRoleByChannel :many
SELECT * FROM roles WHERE channel_name = ?;

-- name: GetRoleByMemberAndChannel :one
SELECT * FROM roles WHERE member_name = ? AND channel_name = ? LIMIT 1;

-- name: GetRoleByTask :one
SELECT r.* FROM roles r
JOIN tasks d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;

-- name: GetRole :one
SELECT * FROM roles WHERE id = ? LIMIT 1;

-- name: GetRoles :many
SELECT * FROM roles;