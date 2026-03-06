-- name: GetBingo :many
SELECT * FROM bingo;

-- name: GetBingoById :one
SELECT * FROM bingo WHERE id = $1;

-- name: GetBingoByEmail :many
SELECT * FROM bingo WHERE created_by_email = $1;

-- name: CreateBingo :one
INSERT INTO bingo (title, description, cells, created_by_email) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateBingo :one
UPDATE bingo SET title = $2, description = $3, cells = $4 WHERE id = $1 RETURNING *;

-- name: DeleteBingo :exec 
DELETE FROM bingo WHERE id = $1;

