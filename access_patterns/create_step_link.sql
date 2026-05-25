-- name: CreateStepLink :one
INSERT INTO step_links (prev_id, next_id, linked_at) VALUES (?, ?, ?) RETURNING *;