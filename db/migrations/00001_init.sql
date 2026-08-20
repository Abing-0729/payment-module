-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
                                        id         BIGSERIAL PRIMARY KEY,
                                        balance    BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- +goose Down
DROP TABLE IF EXISTS accounts;
