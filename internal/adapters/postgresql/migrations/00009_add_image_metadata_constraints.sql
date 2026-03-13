-- +goose Up
-- +goose StatementBegin
ALTER TABLE images
    ALTER COLUMN filename SET NOT NULL,
    ALTER COLUMN fileext SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE images
    ALTER COLUMN filename DROP NOT NULL,
    ALTER COLUMN fileext DROP NOT NULL;
-- +goose StatementEnd
