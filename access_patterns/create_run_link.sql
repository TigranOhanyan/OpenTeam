-- name: CreateRunLink :one
INSERT INTO run_links (parent_run_id, child_run_id, spawning_step_id) VALUES (?, ?, ?) RETURNING *;