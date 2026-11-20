// Package wallet holds Wallet ledger schema artifacts and migration-backed tests.
//
// Runtime schema and sqlc schema input: db/wallet/migrations (golang-migrate;
// sqlc reads *.up.sql and ignores *.down.sql; see make migrate-up / make sqlc).
// Query files: db/wallet/queries.
package wallet
