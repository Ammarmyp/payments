# 04: Post Ledger Transactions atomically

**What to build:** Callers can post a balanced Ledger Transaction with its Entries in one atomic store operation; unbalanced or failed posts never leave partial ledger data; database invariants remain authoritative under concurrency.

**Blocked by:** 03 (persist Accounts).

**Status:** resolved

- [x] sqlc queries CreateTransaction and CreateEntry exist; store Post Ledger Transaction orchestrates them inside the shared in-transaction helper
- [x] inTx begins, commits on success, rolls back on error, rolls back then re-panics on panic, returns commit errors, respects context
- [x] Go pre-validates non-empty and per-Currency debit/credit equality; Postgres deferred checks remain the hard authority; failures map to typed store errors
- [x] Integration test: successful balanced post persists Ledger Transaction and Entries
- [x] Integration test: unbalanced (or empty) post is rejected and nothing is persisted
- [x] Integration test: mid-post constraint/FK failure rolls back completely (no leftover rows by ID)
- [x] Integration test: concurrent valid posts against the same Accounts succeed without corrupting balance invariants (no shared-table truncates; unique IDs)

## Answer

Added sqlc `CreateTransaction` / `CreateEntry` (`db/wallet/queries/ledger.sql`), shared `postgres.InTx` (commit / rollback / re-panic / canceled context), and `Store.PostLedgerTransaction` with Go empty + per-Currency balance pre-checks. Postgres `check_violation` / FK failures map to `ErrEmptyLedgerTransaction`, `ErrUnbalancedLedgerTransaction`, and `ErrLedgerInvariant`. Integration tests cover balanced persist, empty/unbalanced reject with no leftover rows, mid-post FK full rollback, and concurrent posts with derived balance checks. No commit (per request).
