-- name: CreateTaskExecution :one
INSERT INTO task_executions (execution_id, task_id) VALUES (?, ?) RETURNING *;

-- name: GetTaskExecution :one
SELECT * FROM task_executions WHERE execution_id = ? LIMIT 1;

-- name: GetTaskExecutions :many
SELECT * FROM task_executions ORDER BY execution_id ASC;