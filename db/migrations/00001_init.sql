-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
                                        id         BIGSERIAL PRIMARY KEY,
                                        balance    BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );
CREATE TABLE IF NOT EXISTS orders (
                                      id           BIGSERIAL PRIMARY KEY,
                                      order_no     VARCHAR(32) NOT NULL UNIQUE,
    user_id      BIGINT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    status       VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    version      BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
    );
-- +goose Down
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS orders;
