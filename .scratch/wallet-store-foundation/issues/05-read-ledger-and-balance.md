# 05: Read ledger history and Account Balance

**What to build:** After posts, callers can load a Ledger Transaction with its Entries, list an Account’s Entries, and read Account Balance derived only from posted Entries with correct debit/credit semantics relative to Normal Balance—not from a cached column.

**Blocked by:** 04 (post Ledger Transactions atomically).

**Status:** resolved

- [x] sqlc queries GetTransaction, ListEntriesByTransaction, ListEntriesByAccount, GetAccountBalance exist and are used via the Wallet store
- [x] Get Account Balance reads the derived balances view (or equivalent derivation from Entries); no writable/cached balance field on Account
- [x] Integration test: retrieve a posted Ledger Transaction and all of its Entries
- [x] Integration test: list Entries for an Account includes the expected lines after posting
- [x] Integration test: Account Balance matches expected Amount after debit/credit posts given the Account’s Normal Balance (e.g. debit-normal vs credit-normal cases as needed)
- [x] No speculative HTTP read APIs; store + tests only

## Answer

Added sqlc reads `GetTransaction`, `ListEntriesByTransaction`, `ListEntriesByAccount` (`db/wallet/queries/ledger.sql`) and `GetAccountBalance` over `account_balances` (`db/wallet/queries/accounts.sql`). Store exposes `GetLedgerTransaction` (transaction + entries), `ListEntriesByAccount`, and `GetAccountBalance`. Integration tests cover retrieve-with-entries, per-account entry listing, and debit-/credit-normal balance semantics (including opposing Entries). No commit (per request).

## Comments
