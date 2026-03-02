-- +goose Up
-- +goose StatementBegin
ALTER TABLE bingo
    ADD COLUMN card_img_preview BYTEA;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bingo
    DROP COLUMN card_img_preview;
-- +goose StatementEnd
