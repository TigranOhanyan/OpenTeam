-- name: GetPendingStepsByRun :many
SELECT * FROM steps WHERE run_id = ? AND status = 'pending';