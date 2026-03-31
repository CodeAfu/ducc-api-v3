-- +goose Up
-- +goose StatementBegin
ALTER TABLE char_details
    ADD COLUMN display_name VARCHAR(64);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE char_details
    DROP COLUMN IF EXISTS display_name;
-- +goose StatementEnd
