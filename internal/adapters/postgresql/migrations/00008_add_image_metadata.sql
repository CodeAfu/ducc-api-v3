-- +goose Up
-- +goose StatementBegin
ALTER TABLE images
    ADD COLUMN filename VARCHAR(60),
    ADD COLUMN fileext VARCHAR(8);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE images
    DROP COLUMN IF EXISTS filename,
    DROP COLUMN IF EXISTS fileext;
-- +goose StatementEnd
