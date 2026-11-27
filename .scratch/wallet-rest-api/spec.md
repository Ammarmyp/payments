# Wallet REST API: Accounts, deposits, and ledger pages

Status: ready-for-agent

## Problem Statement

The Wallet service can persist Accounts and post balanced Ledger Transactions, but callers have no versioned HTTP API to open an owned Account, read its Account Balance, deposit funds from outside the ledger, or page through its Entries. Without that surface—and without seeded Settlement Accounts per Currency—money cannot enter the ledger in a product-safe, double-entry way.

## Solution

Expose five versioned REST endpoints over a Service layer (validation and rules, no HTTP) backed by the existing Store and PostgreSQL. Add Account Kind and uniqueness rules, seed Settlement Accounts for USD and ETB, post deposits as debit Settlement / credit owned Account (ADR-0002), represent Money in JSON as integer minor units with Currency, and list Entries with opaque keyset cursors. Prove the stack with httptest handler tests against real Postgres and a curl smoke script for a live demo.

## User Stories

1. As an API client, I want to open an owned Account with an Owner and Currency via `POST /v1/wallets`, so that I can hold funds for that Owner in that Currency.
2. As an API client, I want create to return 201 with the Account’s id, owner_id, currency, and created_at, so that I can store the id for later calls.
3. As an API client, I want create to reject a malformed Owner id with a validation error on owner_id, so that I can fix the request.
4. As an API client, I want create to reject a non–ISO-4217 Currency with a validation error on currency, so that I only send real currency codes.
5. As an API client, I want create to reject an unsupported Currency (no Settlement Account) with a validation error on currency, so that I know which Currencies the product funds.
6. As an API client, I want a second create for the same Owner and Currency to return 409 conflict, so that I do not silently get two Accounts.
7. As an API client, I want create to omit any name field, so that I am not responsible for Account naming.
8. As an API client, I want to fetch an owned Account by id via `GET /v1/wallets/{id}`, so that I can confirm it exists.
9. As an API client, I want get to return 404 for an unknown id, so that missing and forbidden-looking resources look the same.
10. As an API client, I want get to return 404 for a System Account id, so that platform ledger positions stay invisible.
11. As an API client, I want a path id that is not a UUID to return 404, so that invalid ids do not look like a different error class than unknown ids.
12. As an API client, I want `GET /v1/wallets/{id}/balance` to return Account Balance as `{amount, currency}` in minor units, so that I never parse floats.
13. As an API client, I want balance on a new Account to be zero in that Account’s Currency, so that empty ledgers are unambiguous.
14. As an API client, I want balance for a missing or System Account to return 404, so that I cannot probe platform Accounts via balance.
15. As an API client, I want to deposit via `POST /v1/wallets/{id}/deposits` with `{amount, currency}`, so that external funds appear on the Owner’s Account.
16. As an API client, I want a successful deposit to return 201 with the Ledger Transaction id and the deposited Money, so that I can correlate the movement without seeing Settlement Entries.
17. As an API client, I want a deposit to increase the owned Account’s Account Balance by the deposited Amount, so that the product narrative matches the ledger.
18. As an API client, I want a deposit Currency that does not match the Account’s Currency to be rejected as validation, so that 500 never quietly becomes 500 of another Currency.
19. As an API client, I want amount less than or equal to zero to be rejected as validation, so that empty or negative deposits never post.
20. As an API client, I want amount above the single-deposit cap to be rejected as validation, so that typos cannot post huge movements.
21. As an API client, I want a deposit against a missing or System Account to return 404, so that I cannot fund platform Accounts through this API.
22. As an API client, I want repeated deposits each to create a new Ledger Transaction, so that behaviour is predictable even without idempotency keys.
23. As an API client, I want `GET /v1/wallets/{id}/entries` to return Entries newest first, so that recent activity is first.
24. As an API client, I want each Entry in the page to include id, direction, amount, currency, transaction_id, and created_at, so that I can audit lines without Account id noise.
25. As an API client, I want an empty Account’s entry list to be an empty array with an empty next_cursor, so that “no history” is simple.
26. As an API client, I want to pass limit (default 20, max 100) and an opaque cursor, so that I can page without offsets.
27. As an API client, I want a non-integer or out-of-range limit to be a validation error on limit, so that bad pagination input is clear.
28. As an API client, I want a malformed cursor to be a validation error on cursor, so that I do not get a 404 for bad paging state.
29. As an API client, I want next_cursor set when more Entries exist and empty on the last page, so that I know when to stop without COUNT(*).
30. As an API client, I want unmatched routes to return a JSON 404, so that clients always get the stable error shape.
31. As an API client, I want all wallet routes under `/v1/`, so that later breaking changes can live under a new version prefix.
32. As a ledger operator, I want deposits funded from a Settlement Account, so that money never appears from nowhere on the ledger.
33. As a ledger operator, I want Settlement Accounts for USD and ETB seeded by migration, so that supported Currencies exist without a create race.
34. As a ledger operator, I want Settlement Accounts to be debit-normal System Accounts named settlement, so that cash-at-bank semantics are consistent (ADR-0002).
35. As a ledger operator, I want owned Accounts created by this API to be credit-normal with Account Kind wallet, so that a credit Entry increases what we owe the Owner.
36. As a ledger operator, I want each deposit to post one balanced Ledger Transaction (debit Settlement, credit owned Account) with description deposit, so that the movement is auditable and append-only (ADR-0001).
37. As a ledger operator, I want PostgreSQL to remain the last line of defence for ledger balance and immutability, so that buggy writers cannot commit invalid or rewritten history.
38. As a domain collaborator, I want Account Kind wallet vs system enforced with Owner nullability, so that owned Accounts always have an Owner and System Accounts never do.
39. As a domain collaborator, I want at most one wallet Account per Owner and Currency, so that CreateWallet is deterministic.
40. As a domain collaborator, I want at most one System Account per name and Currency, so that Settlement lookup by name is deterministic.
41. As a domain collaborator, I want existing rows backfilled to kind from owner_id only, without rewriting historical normal_balance, so that old Entries keep honest Account Balance math.
42. As a Wallet developer, I want validation and business rules in a Service with no HTTP types, so that every entry path gets the same rules.
43. As a Wallet developer, I want the HTTP handler to translate JSON and URLs only, so that DTO shapes stay a client contract separate from store models.
44. As a Wallet developer, I want Store helpers to load a System Account by name and Currency and to detect unique violations, so that Settlement lookup and conflict mapping stay in one place.
45. As a Wallet developer, I want ErrWalletNotFound at the Service boundary and ErrAccountNotFound at the Store, both aligning with not-found for HTTP, so that layers keep clear vocabulary.
46. As a Wallet developer, I want a missing Settlement at deposit time to surface as an internal failure to the client (not 404), so that our seeding bug is not mistaken for a bad client id.
47. As a Wallet developer, I want stdlib ServeMux method+path patterns for the five routes, so that we do not take on a router library.
48. As a Wallet developer, I want shared HTTP helpers for error-returning handlers and integer query parsing, so that wallet handlers and future APIs share one pattern.
49. As a platform operator, I want the Wallet process to register the wallet API beside health/ready and keep JSON 404 for everything else, so that one process serves ops and product routes.
50. As a test engineer, I want handler tests that drive httptest through Service, Store, and real PostgreSQL, so that the public contract is locked end-to-end.
51. As a test engineer, I want tests skipped when TEST_DATABASE_URL is unset, so that unit-only environments still pass.
52. As a test engineer, I want each test to use unique Owner and Account ids without truncating tables, so that tests stay parallel-safe.
53. As a human demoing the API, I want a curl smoke script against make run-wallet, so that I can exercise all five endpoints without a UI.
54. As a future agent, I want glossary and ADR-0002 respected (Account not Wallet-as-entity; owned Accounts are liabilities), so that later work does not invert deposit semantics.

## Implementation Decisions

- Layers: Handler (JSON/URLs) → Service (rules) → Store (SQL/transactions) → PostgreSQL (constraints). Each layer talks only to the one below.
- Domain vocabulary per root glossary and ADR-0001 / ADR-0002: Account, Account Kind, System Account, Settlement Account, Owner, Ledger Transaction, Entry, Normal Balance, Currency, Amount, Account Balance. HTTP resource name “wallet” is not a domain entity.
- Migration on the existing empty wallets migration: add kind; backfill kind from owner_id (null → system, else wallet); CHECKs pairing kind with owner nullability; partial uniques for one wallet per (owner_id, currency) and one system Account per (name, currency); replace Entries index on account_id alone with (account_id, id) for keyset; seed Settlement Accounts for USD and ETB (kind system, name settlement, debit-normal); down migration reverses in order.
- sqlc: extend Account queries for kind; add System Account get-by-name-and-currency; add keyset list Entries by Account (id less than cursor, order by id desc, limit); regenerate via the repo’s sqlc Make target.
- Store: create Account accepts Kind (and related fields); SystemAccount(name, currency); ErrAccountExists on unique violation; helper to detect Postgres unique violations; keep ErrAccountNotFound for missing rows.
- Service constants: SettlementAccount name `settlement`; MaxAmount `100_000_000` minor units; DefaultPageSize 20; MaxPageSize 100; ErrWalletNotFound wrapping not-found.
- Service Money is the (amount, currency) pair for API-facing amounts; glossary term remains Amount.
- CreateWallet validates owner_id and currency as strings; requires Settlement for that currency; creates Kind wallet, credit-normal, Name `currency + " wallet"`; maps ErrAccountExists to conflict. Decision-rich shape from design sketch:

```go
func (s *Service) CreateWallet(ctx context.Context, ownerID, currency string) (Account, error)
// Name: currency + " wallet"; KindWallet; NormalBalance credit;
// unsupported currency when SystemAccount(SettlementAccount, currency) is not found
```

- Wallet/get: load by id; if missing or Kind system → ErrWalletNotFound.
- Balance: derived Account Balance for an owned Account only.
- Deposit: validate amount (>0, ≤ MaxAmount) and currency match; load Settlement; PostLedgerTransaction description `deposit` with debit Settlement and credit owned Account; if Settlement missing after validation, return an internal error to the client (do not wrap as not-found). No idempotency key in this slice.
- Entries: resolve owned Account; decode optional cursor from opaque base64 of Entry id; fetch limit+1; return page + next_cursor when extra row exists; invalid cursor → validation on cursor.
- HTTP DTOs: wallet `{id, owner_id, currency, created_at}`; balance `{amount, currency}`; deposit 201 `{id, amount, currency}` (Ledger Transaction id + Money); entry `{id, direction, amount, currency, transaction_id, created_at}`; entries `{entries, next_cursor}`.
- Routes on stdlib mux: POST/GET wallets, GET balance, POST deposits, GET entries; owner_id trusted from JSON body until auth exists.
- Platform HTTP: add error-returning HandlerFunc and QueryInt; keep existing JSON error mapping (validation 422, not-found 404, conflict 409).
- Process entrypoint: construct Store and Service from the pool; register wallet Handler; retain health/ready and JSON 404 fallback.
- Live demo: curl smoke script exercising the five endpoints against a bootstrapped Wallet process (no demo UI).

## Testing Decisions

- Good tests assert external behaviour only: HTTP status, JSON bodies, and observable ledger outcomes (Account Balance, Entry pages)—not SQL shape, index names, or private helpers.
- Primary seam (one high seam): Wallet HTTP handlers via httptest, wired to real Service + Store + PostgreSQL using TEST_DATABASE_URL (skip if unset). This is the acceptance seam for the feature.
- Cover: create success and validation/conflict; get wallet 200/404 (including System Account id and non-UUID path); balance zero and after deposit; deposit success, currency mismatch, amount bounds, 404 targets; entries empty, paging with cursor, invalid limit/cursor, newest-first ordering; unmatched route JSON 404.
- Isolate data with fresh Owner/Account UUIDs; do not truncate shared tables (same approach as existing Wallet store tests).
- Prior art: Wallet health httptest tests for HTTP style; Wallet store integration tests for Postgres + TEST_DATABASE_URL skipping; platform httpx tests for error JSON shapes.
- Existing Store posting/balance tests remain valuable for ledger mechanics; rewrite fixtures that assumed debit-normal “customer cash” when they would contradict ADR-0002 deposit semantics used by the API. Do not add a separate Service-only test package for this feature unless the HTTP seam cannot express a rule.

## Out of Scope

- Authentication/authorization (Owner is client-supplied; no session binding).
- Deposit idempotency keys or deduplication.
- Withdrawals, transfers, or any non-deposit Ledger Transaction product API.
- Currencies beyond USD and ETB (new Currency = new migration later).
- Client-chosen Account names or exposing name/kind/normal_balance on wallet DTOs.
- Offset pagination, GraphQL, or a third-party HTTP router.
- Demo frontend or mobile UI.
- Changing ADR-0001 append-only / Postgres-enforced balance rules.
- Rewriting normal_balance on pre-existing Account rows in the migration.
- Auth, Transfer, or other bounded contexts’ HTTP APIs.

## Further Notes

- Makefile today exposes `sqlc` for codegen (not a `generate` alias); agents should use the existing target.
- Glossary and ADR-0002 were updated during design grill; implement against those docs.
- Foundation Store APIs (CreateAccount, PostLedgerTransaction, GetAccountBalance, ListEntriesByAccount) already exist; this work extends schema/queries/store and adds Service + HTTP on top rather than replacing the ledger core.
