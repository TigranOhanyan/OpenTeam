-- name: GetRunsBySourceStep :many
SELECT * FROM runs WHERE source_step_id = ?;