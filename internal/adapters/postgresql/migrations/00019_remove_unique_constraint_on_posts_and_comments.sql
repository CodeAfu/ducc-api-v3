-- +goose Up
-- +goose StatementBegin
ALTER TABLE reddit_posts DROP CONSTRAINT IF EXISTS reddit_posts_url_key;
ALTER TABLE reddit_comments DROP CONSTRAINT IF EXISTS reddit_comments_url_key;
ALTER TABLE hyl_posts DROP CONSTRAINT IF EXISTS hyl_posts_url_key;
ALTER TABLE hyl_comments DROP CONSTRAINT IF EXISTS hyl_comments_url_key;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE reddit_posts ADD CONSTRAINT reddit_posts_url_key UNIQUE (url);
ALTER TABLE reddit_comments ADD CONSTRAINT reddit_comments_url_key UNIQUE (url);
ALTER TABLE hyl_posts ADD CONSTRAINT hyl_posts_url_key UNIQUE (url);
ALTER TABLE hyl_comments ADD CONSTRAINT hyl_comments_url_key UNIQUE (url);
-- +goose StatementEnd
