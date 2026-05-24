-- name: GetFirstTask :one
SELECT * FROM tasks
WHERE role_id = ?
AND prev_id IS NULL
LIMIT 1;