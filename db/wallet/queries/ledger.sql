-- name: CreateTransaction :one
INSERT INTO transactions (
    id,
    description
) VALUES (
    $1, $2
)
RETURNING *;

-- name: CreateEntry :one
INSERT INTO entries (
    transaction_id,
    account_id,
    direction,
    amount,
    currency
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetTransaction :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: ListEntriesByTransaction :many
SELECT *
FROM entries
WHERE transaction_id = $1
ORDER BY id;

-- name: ListEntriesByAccount :many
SELECT *
FROM entries
WHERE account_id = $1
ORDER BY created_at, id;
