-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_movies_title ON movies USING GIN (to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS idx_movies_genres ON movies (genres);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_movies_title;

DROP INDEX IF EXISTS idx_movies_genres;

-- +goose StatementEnd
