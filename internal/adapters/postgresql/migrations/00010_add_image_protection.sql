-- +goose Up
-- +goose StatementBegin
ALTER TABLE images ADD COLUMN is_protected BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE images DROP COLUMN is_protected;
-- +goose StatementEnd
