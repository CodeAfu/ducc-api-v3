-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    img_data BYTEA NOT NULL,
    img_hash VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS images;
-- +goose StatementEnd
