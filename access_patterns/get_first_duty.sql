-- name: GetFirstDuty :one
SELECT * FROM duties
WHERE role_id = ?
AND prev_id IS NULL
LIMIT 1;