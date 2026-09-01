-- +goose Up
CREATE TABLE IF NOT EXISTS payment_attempts (
                                                id                  BIGSERIAL PRIMARY KEY,
                                                order_no            VARCHAR(32) NOT NULL,
    amount_cents        BIGINT NOT NULL CHECK (amount_cents > 0),
    channel             VARCHAR(32) NOT NULL,
    idempotency_key     VARCHAR(128) NOT NULL,
    request_fingerprint VARCHAR(64) NOT NULL,
    status              VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT payment_attempts_idempotency_key_unique
    UNIQUE (idempotency_key)
    );

CREATE INDEX IF NOT EXISTS payment_attempts_order_no_idx
    ON payment_attempts(order_no);

-- +goose Down
DROP TABLE IF EXISTS payment_attempts;
