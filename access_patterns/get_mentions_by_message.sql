-- name: GetMentionsByMessageId :one
SELECT m.* FROM mentions m WHERE m.message_id = ?;