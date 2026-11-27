-- name: CreateAccount :one
INSERT INTO accounts (
    id,
    owner_id,
    name,
    currency,
    normal_balance,
    kind
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = $1;

-- name: GetSystemAccount :one
SELECT *
FROM accounts
WHERE kind = 'system'
  AND name = $1
  AND currency = $2;

-- name: ListAccountsByOwner :many
SELECT *
FROM accounts
WHERE owner_id = $1
ORDER BY created_at, id;

-- name: GetAccountBalance :one
SELECT account_id, name, currency, balance
FROM account_balances
WHERE account_id = $1;
