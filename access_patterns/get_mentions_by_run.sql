-- name: GetMentionsByRun :many
SELECT a.* FROM mentions a
WHERE a.run_id = ?
ORDER BY a.created_at ASC;