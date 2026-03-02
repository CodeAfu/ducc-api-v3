-- name: GetBingo :many
SELECT * FROM bingo;

-- name: GetBingoById :one
SELECT * FROM bingo WHERE id = $1;

-- name: CreateBingo :one
INSERT INTO bingo (title, description, cells) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateBingo :one
UPDATE bingo SET title = $2, description = $3, cells = $4 WHERE id = $1 RETURNING *;

-- name: DeleteBingo :exec 
DELETE FROM bingo WHERE id = $1;


-- name: GetImage :one
SELECT img_data FROM images WHERE id = $1;

-- name: CreateImage :one
INSERT INTO images (img_data, img_hash) VALUES ($1, $2) RETURNING *;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = $1;
