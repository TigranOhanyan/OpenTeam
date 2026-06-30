-- name: CreateExecution :one
INSERT INTO executions (id, kind) VALUES (?, ?) RETURNING *;

-- name: CloseExecution :one
UPDATE executions SET status = 'closed' WHERE id = ? RETURNING *;

-- name: CreateExecutionLink :one
INSERT INTO execution_links (parent_id, child_id) VALUES (?, ?) RETURNING *;

-- name: GetAllExecutions :many
SELECT * FROM executions e
ORDER BY e.id ASC;

-- name: GetExecution :one
SELECT * FROM executions WHERE id = ? LIMIT 1;

-- name: GetChildExecutions :many
SELECT 
    child.*
FROM executions AS child
INNER JOIN execution_links AS el ON child.id = el.child_id
WHERE 
    el.parent_id = ?
ORDER BY child.id DESC;

-- name: GetOpenExecutions :many
SELECT * 
FROM executions 
WHERE status = 'open' 
ORDER BY id DESC;