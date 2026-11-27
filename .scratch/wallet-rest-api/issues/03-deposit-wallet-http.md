# 03: Deposit into a wallet over HTTP

**What to build:** A client can add external funds to an owned Account via `POST /v1/wallets/{id}/deposits`, posting a balanced Ledger Transaction (debit Settlement, credit owned Account) so Account Balance increases, with httptest coverage through to Postgres.

**Blocked by:** 02 (Open and read a wallet over HTTP).

**Status:** resolved

- [x] Service Deposit validates amount (>0, ≤ MaxAmount) and matching currency; posts description `deposit` as debit Settlement / credit owned Account (ADR-0002)
- [x] Missing or System Account targets yield ErrWalletNotFound; a missing Settlement after those checks surfaces as an internal client error (not 404)
- [x] Handler returns 201 with Ledger Transaction id and deposited Money; no Settlement Entry in the JSON
- [x] httptest → Postgres tests cover successful deposit, balance increase via GET balance, amount/currency validation, and 404 targets; skip without TEST_DATABASE_URL
- [x] No deposit idempotency key in this ticket

## Answer

Added `Service.Deposit` (`MaxAmount`, owned-Account gate, currency/amount validation, Settlement lookup → internal error if missing, ADR-0002 debit Settlement / credit owned with description `deposit`) and `POST /v1/wallets/{id}/deposits` returning 201 `{id, amount, currency}`. httptest→Postgres covers success, balance increase, amount/currency validation, repeat distinct ids, and 404 for missing/System Account. No commit (per request).
