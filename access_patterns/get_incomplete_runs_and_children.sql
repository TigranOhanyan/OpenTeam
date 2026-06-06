-- name: GetIncompleteRunsAndChildren :many
SELECT 
    r.id AS run_id,
    r.status AS run_status,
    r.kind AS run_kind,
    rl.child_run_id AS child_run_id,
    child.status AS child_status
FROM runs r
LEFT JOIN run_links rl ON r.id = rl.parent_run_id
LEFT JOIN runs child ON rl.child_run_id = child.id
WHERE r.status != 'completed';