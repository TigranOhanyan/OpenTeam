-- name: GetRoleByDuty :one
SELECT r.* FROM roles r
JOIN duties d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;