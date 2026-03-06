-- name: GetImages :many
SELECT * FROM images;

-- name: GetImageById :one
SELECT img_data FROM images WHERE id = $1;

-- name: CreateImage :one
INSERT INTO images (img_data, img_hash, added_by) VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = $1;

