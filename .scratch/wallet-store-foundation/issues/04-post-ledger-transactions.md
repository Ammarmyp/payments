# 04: Post Ledger Transactions atomically

**What to build:** Callers can post a balanced Ledger Transaction with its Entries in one atomic store operation; unbalanced or failed posts never leave partial ledger data; database invariants remain authoritative under concurrency.

**Blocked by:** 03 (persist Accounts).

**Status:** ready-for-agent

- [ ] sqlc queries CreateTransaction and CreateEntry exist; store Post Ledger Transaction orchestrates them inside the shared in-transaction helper
- [ ] inTx begins, commits on success, rolls back on error, rolls back then re-panics on panic, returns commit errors, respects context
- [ ] Go pre-validates non-empty and per-Currency debit/credit equality; Postgres deferred checks remain the hard authority; failures map to typed store errors
- [ ] Integration test: successful balanced post persists Ledger Transaction and Entries
- [ ] Integration test: unbalanced (or empty) post is rejected and nothing is persisted
- [ ] Integration test: mid-post constraint/FK failure rolls back completely (no leftover rows by ID)
- [ ] Integration test: concurrent valid posts against the same Accounts succeed without corrupting balance invariants (no shared-table truncates; unique IDs)
