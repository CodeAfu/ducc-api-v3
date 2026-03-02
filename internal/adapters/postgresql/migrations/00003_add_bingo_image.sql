-- +goose Up
-- +goose StatementBegin
ALTER TABLE bingo 
    ADD COLUMN img_center BIGINT REFERENCES images(id) ON DELETE SET NULL,
    ADD COLUMN img_bg BIGINT REFERENCES images(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bingo 
    DROP COLUMN IF EXISTS img_center,
    DROP COLUMN IF EXISTS img_bg;
-- +goose StatementEnd
