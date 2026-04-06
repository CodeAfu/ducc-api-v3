-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS hyl_scrape_session (
    id BIGSERIAL PRIMARY KEY,
    target TEXT NOT NULL,
    created_by_email TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hyl_posts (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES hyl_scrape_session(id) ON DELETE CASCADE,
    url TEXT UNIQUE NOT NULL,
    author TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hyl_comments (
    id BIGSERIAL PRIMARY KEY,    
    session_id BIGINT NOT NULL REFERENCES hyl_scrape_session(id) ON DELETE CASCADE,
    post_id BIGINT NOT NULL REFERENCES hyl_posts(id) ON DELETE CASCADE,
    parent_comment_id BIGINT REFERENCES hyl_comments(id) ON DELETE CASCADE,  -- for nested replies
    url TEXT UNIQUE NOT NULL,
    author TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS hyl_comments;
DROP TABLE IF EXISTS hyl_posts;
DROP TABLE IF EXISTS hyl_scrape_session;
-- +goose StatementEnd
