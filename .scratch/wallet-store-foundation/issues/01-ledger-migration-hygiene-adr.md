# 01: Prefactor: ledger migration hygiene + ADR

**What to build:** Correct the initial ledger migration so database-enforced Ledger Transaction balance/non-empty invariants and append-only behaviour are trustworthy, record that decision in an ADR, and prove those invariants with migration-backed tests on a clean database (volume wipe accepted).

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] Initial migration fixes are applied in place (balance-trigger table-name branch, Ledger Transaction created-at default, known comment cleanups)
- [x] `migrate-up` succeeds on a wiped/clean Wallet database
- [x] Tests (or migration-backed checks) verify an empty or unbalanced Ledger Transaction cannot commit
- [x] Tests verify UPDATE/DELETE on Ledger Transactions and Entries are rejected (append-only)
- [x] ADR records Postgres-enforced ledger invariants and append-only corrections via reversing Ledger Transactions
- [x] Temporary invalid SQL or debug artifacts are not left behind

## Answer

Fixed `000001_init_ledger` in place: `TG_TABLE_NAME = 'transactions'`, deferred constraint trigger on `transactions` (so empty posts fail at commit), `created_at DEFAULT now()`, comment cleanups. Added ADR `docs/adr/0001-postgres-enforced-ledger-invariants.md`. Migration-backed tests in `db/wallet` cover empty/unbalanced reject, balanced accept, append-only on transactions/entries, and created_at default. Verified with volume wipe + `make migrate-up` + `go test ./db/wallet`.
