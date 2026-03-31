-- +goose Up
-- +goose StatementBegin
ALTER TABLE profile_chars
    ADD COLUMN asc_level SMALLINT NOT NULL DEFAULT 20
    CONSTRAINT check_asc_level_range CHECK (asc_level >= 0 AND asc_level <= 100);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profile_chars
    DROP COLUMN IF EXISTS asc_level;
-- +goose StatementEnd
