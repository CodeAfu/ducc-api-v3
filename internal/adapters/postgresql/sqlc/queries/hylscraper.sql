-- name: CreateHylScrapeSession :one
INSERT INTO hyl_scrape_session (created_by_email)
    VALUES ($1) RETURNING *;

-- name: GetHylScrapeSessionByEmail :many
SELECT * FROM hyl_scrape_session WHERE created_by_email = $1;

-- name: GetHylScrapeSessionById :one
SELECT * FROM hyl_scrape_session WHERE id = $1;

-- name: ClaimHylScrapeSession :one
UPDATE hyl_scrape_session
SET scrape_begin = NOW()
WHERE id = $1
  AND created_by_email = $2
  AND scrape_begin IS NULL
RETURNING *;

-- name: AddScrapeErrorById :one
UPDATE hyl_scrape_session
SET errors = COALESCE(errors, '{}'::TEXT[]) || $2::TEXT[]
WHERE id = $1
RETURNING *;

-- name: UpdateHylScraperSession :one
UPDATE hyl_scrape_session
SET
    description = COALESCE(sqlc.narg(description), description),
    scrape_begin = COALESCE(sqlc.narg(scrape_begin), scrape_begin),
    scrape_end = COALESCE(sqlc.narg(scrape_end), scrape_end)
WHERE id = sqlc.arg(id)
RETURNING *;


-- name: GetHylPostByAuthor :many
SELECT * FROM hyl_posts WHERE author = $1;

-- name: AddHylPost :one
INSERT INTO hyl_posts (session_id, url, author, title, content) 
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *;


-- name: GetHylCommentByAuthor :many
SELECT * FROM hyl_comments WHERE author = $1;

-- name: AddHylComment :one
INSERT INTO hyl_comments (session_id, post_id, parent_comment_id, url, author, content) 
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *;

-- name: AddHylComments :many
INSERT INTO hyl_comments (session_id, post_id, url, author, content)
SELECT
    unnest(sqlc.arg(session_id)::bigint[]),
    unnest(sqlc.arg(post_id)::bigint[]),
    unnest(sqlc.arg(url)::text[]),
    unnest(sqlc.arg(author)::text[]),
    unnest(sqlc.arg(content)::text[])
RETURNING *;


-- name: GetHylPostsBySession :many
SELECT * FROM hyl_posts WHERE session_id = $1;

-- name: GetHylCommentsAndPostsFromAuthor :many
SELECT p.id, p.session_id, p.url, p.author, p.title, p.content, p.created_at, p.updated_at 
    FROM hyl_posts p
    WHERE p.author = $1
UNION
SELECT c.id, c.session_id, c.url, c.author, NULL::TEXT AS title, c.content, c.created_at, c.updated_at 
    FROM hyl_comments c
    WHERE c.author = $1;
