-- name: GetRunsBySpawningStepId :many
SELECT * FROM runs r
INNER JOIN run_links rl ON r.id = rl.child_run_id
WHERE rl.spawning_step_id = ?;