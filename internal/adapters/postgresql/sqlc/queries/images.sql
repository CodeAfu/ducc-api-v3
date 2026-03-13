-- name: GetImages :many
SELECT * FROM images;

-- name: GetImageById :one
SELECT * FROM images WHERE id = $1;

-- name: CreateImage :one
INSERT INTO images (img_data, img_hash, added_by, filename, fileext) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = $1;

