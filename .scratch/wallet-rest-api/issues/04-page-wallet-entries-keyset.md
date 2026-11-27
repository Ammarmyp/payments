# 04: Page wallet Entries with keyset cursors

**What to build:** A client can list an owned Account’s Entries newest first via `GET /v1/wallets/{id}/entries` using opaque cursors and a limit, without offsets or COUNT(*), with httptest coverage through to Postgres (after deposits so real pages exist).

**Blocked by:** 03 (Deposit into a wallet over HTTP).

**Status:** resolved

- [x] sqlc/store can list Entries for an Account with keyset filter on Entry id, order by id descending, and a limit
- [x] Service Entries uses DefaultPageSize 20 / MaxPageSize 100, fetches limit+1 for next_cursor, encodes/decodes opaque base64 Entry id cursors; invalid cursor → validation on cursor
- [x] Platform HTTP QueryInt supports limit parsing with validation errors for bad values
- [x] Handler returns entries + next_cursor; each Entry exposes id, direction, amount, currency, transaction_id, created_at
- [x] httptest → Postgres tests cover empty list, multi-page cursor walk after deposits, newest-first order, invalid limit/cursor, and 404 for missing/System Account; skip without TEST_DATABASE_URL

## Answer

Added sqlc `ListEntriesByAccountKeyset` + Store keyset list (id DESC, optional before_id, limit). Service `Entries` owns DefaultPageSize/MaxPageSize (0→default, clamp validation), limit+1 paging, opaque base64 Entry-id cursors. Platform `httpx.QueryInt` for limit parse/range. Handler `GET /v1/wallets/{id}/entries` returns `{entries, next_cursor}` without account_id. httptest covers empty, newest-first, multi-page walk, invalid limit/cursor, 404 targets; QueryInt unit-tested. No commit (per request).
