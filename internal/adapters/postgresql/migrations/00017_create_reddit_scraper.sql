-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reddit_scrape_session (
    id BIGSERIAL PRIMARY KEY,
    target TEXT NOT NULL,
    created_by_email TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reddit_posts (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES reddit_scrape_session(id) ON DELETE CASCADE,
    url TEXT UNIQUE NOT NULL,
    author TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reddit_comments (
    id BIGSERIAL PRIMARY KEY,    
    session_id BIGINT NOT NULL REFERENCES reddit_scrape_session(id) ON DELETE CASCADE,
    post_id BIGINT NOT NULL REFERENCES reddit_posts(id) ON DELETE CASCADE,
    parent_comment_id BIGINT REFERENCES reddit_comments(id) ON DELETE CASCADE,  -- for nested replies
    url TEXT UNIQUE NOT NULL,
    author TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_reddit_scrape_session_modtime
BEFORE UPDATE ON reddit_scrape_session
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reddit_posts_modtime
BEFORE UPDATE ON reddit_posts
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reddit_comments_modtime
BEFORE UPDATE ON reddit_comments
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd
    
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_reddit_comments_modtime ON reddit_comments;
DROP TRIGGER IF EXISTS update_reddit_posts_modtime ON reddit_posts;
DROP TRIGGER IF EXISTS update_reddit_scrape_session_modtime ON reddit_scrape_session;

DROP TABLE IF EXISTS reddit_comments;
DROP TABLE IF EXISTS reddit_posts;
DROP TABLE IF EXISTS reddit_scrape_session;
-- +goose StatementEnd
