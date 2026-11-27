# 01: Account Kind, Settlement seeds, and store lookups

**What to build:** The ledger schema and Wallet store understand Account Kind, enforce Owner/kind honesty and uniqueness, expose Settlement Account lookup by name and Currency, and ship seeded Settlement Accounts for USD and ETB so later API work can open owned Accounts and fund deposits without racing to create system rows.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] Migration adds Account Kind, backfills from owner_id only (no normal_balance rewrites), CHECKs pairing kind with Owner nullability, and partial uniques (one wallet per Owner+Currency; one System Account per name+Currency)
- [x] Migration replaces the Entries index on account_id alone with a composite suitable for keyset listing by Account and Entry id
- [x] Migration seeds Settlement Accounts for USD and ETB (kind system, name settlement, debit-normal); down migration reverses seeds and schema changes in order
- [x] sqlc Account create/read includes Kind; System Account can be loaded by name and Currency; codegen regenerated via the repo sqlc target
- [x] Store create accepts Kind; SystemAccount returns Settlement (or any System Account) by name and Currency; unique violations map to ErrAccountExists; ErrAccountNotFound remains for missing rows
- [x] Existing Store integration tests stay green with Kind (and ADR-0002-friendly normals where fixtures represent owned Accounts / deposits); tests still skip without TEST_DATABASE_URL and isolate via unique IDs

## Answer

Filled `000002_wallets` up/down: `account_kind`, kind↔owner CHECK, wallet/system partial uniques, `entries (account_id, id)` index, seeded `settlement` USD+ETB (debit-normal). sqlc `CreateAccount` includes Kind; added `GetSystemAccount`. Store: `KindWallet`/`KindSystem`, `SystemAccount`, `ErrAccountExists` (wraps conflict), `isUniqueViolation`. Store and invariants fixtures aligned with ADR-0002 (credit-normal owned / debit-normal system). `make test` green. No commit (per request).
