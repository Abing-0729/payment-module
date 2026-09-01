
-- name: CreatePaymentAttempt :one
INSERT INTO payment_attempts (
    order_no,
    amount_cents,
    channel,
    idempotency_key,
    request_fingerprint,
    status
)
VALUES (
           @order_no,
           @amount_cents,
           @channel,
           @idempotency_key,
           @request_fingerprint,
           'PENDING'
       )
    RETURNING *;

-- name: GetPaymentAttemptByIdempotencyKey :one
SELECT *
FROM payment_attempts
WHERE idempotency_key = @idempotency_key;

-- name: GetPaymentAttemptById :one
SELECT *
FROM payment_attempts
WHERE id = @id;
