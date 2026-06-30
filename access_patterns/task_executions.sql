-- name: CreateTaskExecution :one
INSERT INTO task_executions (id, execution_id) VALUES (?, ?) RETURNING *;

-- name: GetTaskExecution :one
SELECT * FROM task_executions WHERE id = ? LIMIT 1;

-- name: GetTaskExecutionsByExecution :many
SELECT * FROM task_executions WHERE execution_id = ? ORDER BY id ASC;

-- name: GetTaskExecutions :many
SELECT * FROM task_executions ORDER BY id ASC;