-- name: GetStepsByRunId :many
SELECT id, run_id, task_id, status, created_at FROM steps WHERE run_id = ? ORDER BY id ASC;