-- name: GetRoleByTask :one
SELECT r.* FROM roles r
JOIN tasks d ON d.role_id = r.id
WHERE d.id = ?
LIMIT 1;