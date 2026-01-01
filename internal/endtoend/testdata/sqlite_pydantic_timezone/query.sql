-- name: GetEvent :one
SELECT * FROM events
WHERE id = ?;

-- name: ListEvents :many
SELECT * FROM events
ORDER BY created_at;
