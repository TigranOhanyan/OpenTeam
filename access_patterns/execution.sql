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

-- name: GetResolvableExecutionCandidates :many
SELECT 
    parent.*
FROM executions parent
INNER JOIN execution_links el ON parent.id = el.parent_id
INNER JOIN executions child ON el.child_id = child.id
WHERE 
    parent.status = 'open' AND
    child.status = 'closed';

-- name: GetOpenFrontier :many
SELECT parent.* FROM executions AS parent 
LEFT JOIN execution_links AS el ON parent.id = el.parent_id
WHERE 
    parent.status = 'open'  AND
    el.child_id IS NULL
ORDER BY parent.created_at ASC;

-- name: GetChildExecutions :many
SELECT 
    child.*
FROM executions AS child
INNER JOIN execution_links AS el ON child.id = el.child_id
WHERE 
    el.parent_id = ?
ORDER BY child.id DESC;

    
-- name: GetLatestChildExecution :one
SELECT 
    child.*
FROM executions AS child
INNER JOIN execution_links AS el ON child.id = el.child_id
WHERE 
    el.parent_id = ?
ORDER BY child.id DESC
LIMIT 1;

-- name: GetOpenExecutions :many
SELECT * 
FROM executions 
WHERE status = 'open' 
ORDER BY created_at DESC;