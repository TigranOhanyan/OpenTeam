-- name: CreateMember :one
INSERT INTO members (name, kind) VALUES (?, ?) RETURNING *;

-- name: GetMemberByRole :one
SELECT m.name, m.kind FROM members m
JOIN roles r ON r.member_name = m.name
WHERE r.id = ?
LIMIT 1;

-- name: GetMemberByTask :one
SELECT m.* FROM members m
JOIN roles r ON r.member_name = m.name
JOIN tasks d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;

-- name: GetMember :one
SELECT * FROM members WHERE name = ? LIMIT 1;

-- name: GetMembers :many
SELECT * FROM members;