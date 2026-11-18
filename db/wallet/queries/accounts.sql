-- name: CreateAccount :one
INSERT INTO accounts (
    id,
    owner_id,
    name,
    currency,
    normal_balance
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = $1;

-- name: ListAccountsByOwner :many
SELECT *
FROM accounts
WHERE owner_id = $1
ORDER BY created_at, id;
