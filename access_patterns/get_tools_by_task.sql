-- name: GetToolsByTask :many
SELECT * FROM tools WHERE task_id = ?;