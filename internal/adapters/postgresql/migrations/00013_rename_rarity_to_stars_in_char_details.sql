-- +goose Up
-- +goose StatementBegin
ALTER TABLE char_details RENAME COLUMN rarity TO stars;
ALTER TABLE genshin_acc_details RENAME TO genshin_profiles;
ALTER TABLE acc_chars RENAME TO profile_chars;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE char_details RENAME COLUMN stars TO rarity;
ALTER TABLE profile_chars RENAME TO acc_chars;
ALTER TABLE genshin_profiles RENAME TO genshin_acc_details;
-- +goose StatementEnd
