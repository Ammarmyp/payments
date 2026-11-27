# 04: Page wallet Entries with keyset cursors

**What to build:** A client can list an owned Account’s Entries newest first via `GET /v1/wallets/{id}/entries` using opaque cursors and a limit, without offsets or COUNT(*), with httptest coverage through to Postgres (after deposits so real pages exist).

**Blocked by:** 03 (Deposit into a wallet over HTTP).

**Status:** ready-for-agent

- [ ] sqlc/store can list Entries for an Account with keyset filter on Entry id, order by id descending, and a limit
- [ ] Service Entries uses DefaultPageSize 20 / MaxPageSize 100, fetches limit+1 for next_cursor, encodes/decodes opaque base64 Entry id cursors; invalid cursor → validation on cursor
- [ ] Platform HTTP QueryInt supports limit parsing with validation errors for bad values
- [ ] Handler returns entries + next_cursor; each Entry exposes id, direction, amount, currency, transaction_id, created_at
- [ ] httptest → Postgres tests cover empty list, multi-page cursor walk after deposits, newest-first order, invalid limit/cursor, and 404 for missing/System Account; skip without TEST_DATABASE_URL
