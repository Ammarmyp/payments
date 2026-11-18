# 03: Persist Accounts (sqlc + store + integration tests)

**What to build:** Operators/developers can create and read Accounts (user-owned and System Accounts) through the Wallet store, backed by sqlc against the canonical schema snapshot, with a reproducible generation workflow and real-Postgres integration tests.

**Blocked by:** 01 (ledger migration hygiene + ADR), 02 (postgres pool, ready, shutdown).

**Status:** ready-for-agent

- [ ] sqlc is configured for PostgreSQL with the pgx/v5 interface; canonical schema snapshot and Account query files exist; generation is reproducible and documented (e.g. make/script target)
- [ ] Generated code is never hand-edited; a deliberate invalid-query check was run then restored during implementation
- [ ] sqlc Account queries: CreateAccount, GetAccount, ListAccountsByOwner
- [ ] Wallet store creates user Accounts with Owner UUID and System Accounts with null Owner; app-generated UUIDs; Amounts/Currency/Normal Balance rules respected at create
- [ ] Integration tests (skip if test database URL unset) cover user Account create, System Account null Owner, get, and list-by-owner using unique IDs without truncating shared tables
- [ ] Required Go dependencies for persistence (pgx, uuid) are added and compatible with generated code
