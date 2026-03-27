-- +goose Up
-- +goose StatementBegin
ALTER TABLE profile_chars RENAME COLUMN acc_id TO prof_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profile_chars RENAME COLUMN prof_id TO acc_id;
-- +goose StatementEnd

