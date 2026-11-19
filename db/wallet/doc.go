// Package wallet holds Wallet ledger schema artifacts and migration-backed tests.
//
// Runtime schema: db/wallet/migrations (golang-migrate; see make migrate-up).
// sqlc codegen:   db/wallet/schema.sql + db/wallet/queries (see make sqlc).
// Keep the schema snapshot aligned with migrations by hand.
package wallet
