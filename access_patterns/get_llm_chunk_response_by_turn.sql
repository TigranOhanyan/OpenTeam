-- name: GetLlmChunkResponseByTurn :many
SELECT * FROM llm_chunk_responses WHERE turn_id = ? ORDER BY sequence_number ASC;