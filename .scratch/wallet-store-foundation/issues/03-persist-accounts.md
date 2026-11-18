# 03: Persist Accounts (sqlc + store + integration tests)

**What to build:** Operators/developers can create and read Accounts (user-owned and System Accounts) through the Wallet store, backed by sqlc against the canonical schema snapshot, with a reproducible generation workflow and real-Postgres integration tests.

**Blocked by:** 01 (ledger migration hygiene + ADR), 02 (postgres pool, ready, shutdown).

**Status:** resolved

- [x] sqlc is configured for PostgreSQL with the pgx/v5 interface; canonical schema snapshot and Account query files exist; generation is reproducible and documented (e.g. make/script target)
- [x] Generated code is never hand-edited; a deliberate invalid-query check was run then restored during implementation
- [x] sqlc Account queries: CreateAccount, GetAccount, ListAccountsByOwner
- [x] Wallet store creates user Accounts with Owner UUID and System Accounts with null Owner; app-generated UUIDs; Amounts/Currency/Normal Balance rules respected at create
- [x] Integration tests (skip if test database URL unset) cover user Account create, System Account null Owner, get, and list-by-owner using unique IDs without truncating shared tables
- [x] Required Go dependencies for persistence (pgx, uuid) are added and compatible with generated code

## Answer

Added `sqlc.yaml` (PostgreSQL + pgx/v5 → `internal/wallet/db`), canonical snapshot `db/wallet/schema.sql`, Account queries in `db/wallet/queries/accounts.sql`, and `make sqlc`. Confirmed `sqlc generate` fails on an invalid relation then restored queries. Hand-written `internal/wallet.Store` creates Owner-linked and System Accounts (null Owner), requires app-generated UUIDs, and validates Currency / Normal Balance at create. Integration tests in `internal/wallet/store_account_test.go` skip without `TEST_DATABASE_URL` and cover create/get/list-by-owner with unique IDs. pgx + uuid already in `go.mod`. No commit (per request).
