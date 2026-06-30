-- name: CreateReactLoop :one
INSERT INTO react_loops (execution_id, task_id) VALUES (?, ?) RETURNING *;

-- name: GetReactLoop :one
SELECT * FROM react_loops WHERE execution_id = ? LIMIT 1;


-- name: GetReactLoops :many
SELECT * FROM react_loops ORDER BY execution_id ASC;

-- name: UpdateReactLoopStatus :one
UPDATE react_loops SET status = ? WHERE execution_id = ? RETURNING *;
