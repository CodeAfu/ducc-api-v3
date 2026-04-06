-- +goose Up
-- +goose StatementBegin
ALTER TABLE profile_chars
ADD COLUMN talent_na_boosted BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN talent_e_boosted BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN talent_q_boosted BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profile_chars
DROP COLUMN talent_na_boosted,
DROP COLUMN talent_e_boosted,
DROP COLUMN talent_q_boosted;
-- +goose StatementEnd
