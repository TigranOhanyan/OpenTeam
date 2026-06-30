-- name: CreateReactLoop :one
INSERT INTO react_loops (id, execution_id, task_id) VALUES (?, ?, ?) RETURNING *;

-- name: GetReactLoop :one
SELECT * FROM react_loops WHERE id = ? LIMIT 1;

-- name: GetReactLoopByExecution :one
SELECT * FROM react_loops WHERE execution_id = ? ORDER BY id ASC;

-- name: GetReactLoops :many
SELECT * FROM react_loops ORDER BY id ASC;

-- name: UpdateReactLoopStatus :one
UPDATE react_loops SET status = ? WHERE id = ? RETURNING *;
