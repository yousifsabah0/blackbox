-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS tokens (
  hash bytea PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
  expiry timestamp(0) with time zone NOT NULL,
  scope text NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS tokens;

-- +goose StatementEnd
