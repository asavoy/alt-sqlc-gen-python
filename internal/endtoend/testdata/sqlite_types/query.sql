-- name: GetFile :one
SELECT * FROM files
WHERE id = ?;

-- name: ListFiles :many
SELECT * FROM files
ORDER BY name;
