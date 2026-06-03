-- name: GetMemberByRole :one
SELECT m.name, m.kind FROM members m
JOIN roles r ON r.member_name = m.name
WHERE r.id = ?
LIMIT 1;