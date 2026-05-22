-- name: CreateTurnLink :one
INSERT INTO turn_links (prev_id, next_id, linked_at) VALUES (?, ?, ?) RETURNING *;