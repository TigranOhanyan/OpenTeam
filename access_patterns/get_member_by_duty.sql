-- name: GetMemberByDuty :one
SELECT m.* FROM members m
JOIN roles r ON r.member_name = m.name
JOIN duties d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;