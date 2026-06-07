-- name: GetParentRunLink :one
SELECT * FROM run_links WHERE child_run_id = ? LIMIT 1;