-- name: GetActionByRun :one
SELECT * FROM actions WHERE run_id = ? LIMIT 1;