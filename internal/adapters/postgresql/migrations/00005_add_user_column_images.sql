-- +goose Up
-- +goose StatementBegin
ALTER TABLE images ADD COLUMN added_by VARCHAR(64);
CREATE INDEX idx_images_added_by ON images(added_by);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_images_added_by;
ALTER TABLE images DROP COLUMN added_by;
-- +goose StatementEnd
