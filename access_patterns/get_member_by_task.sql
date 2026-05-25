-- name: GetMemberByTask :one
SELECT m.* FROM members m
JOIN roles r ON r.member_name = m.name
JOIN tasks d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;