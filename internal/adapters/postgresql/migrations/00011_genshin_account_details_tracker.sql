-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS elements (
    id SMALLSERIAL PRIMARY KEY,
    name VARCHAR(10) NOT NULL UNIQUE,
    icon_url VARCHAR(255) -- genshindev API
);

INSERT INTO elements (name) VALUES
    ('pyro'),
    ('hydro'),
    ('electro'),
    ('cryo'),
    ('anemo'),
    ('dendro'),
    ('geo');

CREATE TABLE IF NOT EXISTS genshin_acc_details (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS char_details (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    element_id SMALLINT NOT NULL REFERENCES elements(id),
    rarity SMALLINT NOT NULL CHECK (rarity BETWEEN 1 AND 5) DEFAULT 1,
    icon BYTEA,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS acc_chars (
    acc_id BIGINT NOT NULL REFERENCES genshin_acc_details(id) ON DELETE CASCADE,
    char_id BIGINT NOT NULL REFERENCES char_details(id) ON DELETE CASCADE,
    level SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 100) DEFAULT 1,
    constellation SMALLINT NOT NULL CHECK (constellation BETWEEN 0 AND 6) DEFAULT 0,
    talent_NA SMALLINT NOT NULL CHECK (talent_NA BETWEEN 1 AND 15) DEFAULT 1,
    talent_E SMALLINT NOT NULL CHECK (talent_E BETWEEN 1 AND 15) DEFAULT 1,
    talent_Q SMALLINT NOT NULL CHECK (talent_Q BETWEEN 1 AND 15) DEFAULT 1,
    notes TEXT,
    PRIMARY KEY (acc_id, char_id)
);

CREATE INDEX idx_acc_chars_char_id ON acc_chars(char_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_acc_chars_char_id;
DROP TABLE IF EXISTS acc_chars;
DROP TABLE IF EXISTS char_details;
DROP TABLE IF EXISTS genshin_acc_details;
DROP TABLE IF EXISTS elements;
-- +goose StatementEnd
