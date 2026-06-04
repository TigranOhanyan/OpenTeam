-- name: GetMentionByRun :one
SELECT a.* FROM mentions a
WHERE a.run_id = ?
LIMIT 1;