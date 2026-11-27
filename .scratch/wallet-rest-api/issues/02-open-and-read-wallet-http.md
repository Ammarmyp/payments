# 02: Open and read a wallet over HTTP

**What to build:** A client can create an owned Account and read it and its Account Balance over versioned HTTP (`POST /v1/wallets`, `GET /v1/wallets/{id}`, `GET /v1/wallets/{id}/balance`), with validation and rules in a Service layer and httptest coverage through to Postgres.

**Blocked by:** 01 (Account Kind, Settlement seeds, and store lookups).

**Status:** resolved

- [x] Service CreateWallet validates owner_id and currency strings, requires a Settlement Account for the Currency, creates Kind wallet credit-normal with Name `currency + " wallet"`, and maps duplicates to conflict
- [x] Service Wallet and Balance return owned Accounts only; missing ids and System Accounts yield ErrWalletNotFound
- [x] Platform HTTP gains an error-returning HandlerFunc used by wallet routes
- [x] Handler exposes create/get/balance with wallet and Money JSON DTOs (no name/kind/normal_balance on wallet responses); non-UUID path ids are 404
- [x] Wallet process registers the wallet Handler beside health/ready and keeps JSON 404 for unmatched routes
- [x] httptest → Postgres tests cover create success, validation, conflict, get 200/404 (including System Account id), and zero balance; skip without TEST_DATABASE_URL

## Answer

Added `wallet.Service` (CreateWallet / Wallet / Balance with Settlement gate, credit-normal owned Accounts, `ErrWalletNotFound`) and HTTP Handler for `POST/GET /v1/wallets` and balance. Platform `httpx.HandlerFunc` maps returned errors to JSON. Wired in `cmd/wallet` beside health/ready. httptest→Postgres covers create/validation/conflict/get/balance/unmatched 404. No commit (per request).
