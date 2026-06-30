-- name: CreateTask :one
INSERT INTO tasks (id, role_id, instruction) VALUES (?, ?, ?) RETURNING id, role_id, instruction;

-- name: CreateTaskLink :one
INSERT INTO task_links (parent_id, child_id) VALUES (?, ?) RETURNING parent_id, child_id;

-- name: GetFirstTask :one
SELECT * FROM tasks AS t
LEFT JOIN task_links AS tl ON t.id = tl.child_id
WHERE t.role_id = ?
AND tl.parent_id IS NULL
LIMIT 1;

-- name: GetTasks :many
SELECT * FROM tasks;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? LIMIT 1;
