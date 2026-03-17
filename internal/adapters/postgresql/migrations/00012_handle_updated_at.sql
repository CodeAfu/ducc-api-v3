-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_genshin_acc_details_modtime
BEFORE UPDATE ON genshin_acc_details
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_char_details_modtime
BEFORE UPDATE ON char_details
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bingo_modtime
BEFORE UPDATE ON bingo
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hyl_comments_modtime
BEFORE UPDATE ON hyl_comments
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hyl_posts_modtime
BEFORE UPDATE ON hyl_posts
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hyl_scrape_session_modtime
BEFORE UPDATE ON hyl_scrape_session
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_images_modtime
BEFORE UPDATE ON images
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_images_modtime ON images;
DROP TRIGGER IF EXISTS update_hyl_scrape_session_modtime ON hyl_scrape_session;
DROP TRIGGER IF EXISTS update_hyl_posts_modtime ON hyl_posts;
DROP TRIGGER IF EXISTS update_hyl_comments_modtime ON hyl_comments;
DROP TRIGGER IF EXISTS update_bingo_modtime ON bingo;
DROP TRIGGER IF EXISTS update_char_details_modtime ON char_details;
DROP TRIGGER IF EXISTS update_genshin_acc_details_modtime ON genshin_acc_details;

DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose StatementEnd
