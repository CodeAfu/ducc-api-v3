-- +goose Up
-- +goose StatementBegin
ALTER TABLE hyl_scrape_session
    DROP COLUMN target,
    ADD COLUMN scrape_begin TIMESTAMPTZ,
    ADD COLUMN scrape_end TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE hyl_scrape_session
    ADD COLUMN target TEXT NOT NULL,
    DROP COLUMN scrape_begin,
    DROP COLUMn scrape_end;
-- +goose StatementEnd
