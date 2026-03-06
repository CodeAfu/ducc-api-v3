-- +goose Up
-- +goose StatementBegin
ALTER TABLE bingo ADD COLUMN created_by_email TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bingo DROP COLUMN created_by_email;
-- +goose StatementEnd
