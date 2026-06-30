-- name: CreateTask :one
INSERT INTO tasks (id, role_id, prev_id, instruction) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetFirstTask :one
SELECT * FROM tasks
WHERE role_id = ?
AND prev_id IS NULL
LIMIT 1;

-- name: GetTasks :many
SELECT * FROM tasks;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? LIMIT 1;
