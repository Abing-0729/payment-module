-- name: CreateOrder :one
INSERT INTO orders (order_no, user_id, amount_cents, status)
VALUES (@order_no, @user_id, @amount_cents, @status)
RETURNING id, order_no, user_id, amount_cents, status, version, created_at, updated_at;

-- name: GetOrderByOrderNo :one
SELECT id, order_no, user_id, amount_cents, status, version, created_at, updated_at
FROM orders
WHERE order_no = @order_no
LIMIT 1;

-- name: TransitionOrder :execrows
UPDATE orders
SET status = @to_status, version = version + 1, updated_at = now()
WHERE order_no = @order_no AND status = ANY(@from_status::varchar[]);
