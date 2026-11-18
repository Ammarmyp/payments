# 05: Read ledger history and Account Balance

**What to build:** After posts, callers can load a Ledger Transaction with its Entries, list an Account’s Entries, and read Account Balance derived only from posted Entries with correct debit/credit semantics relative to Normal Balance—not from a cached column.

**Blocked by:** 04 (post Ledger Transactions atomically).

**Status:** ready-for-agent

- [ ] sqlc queries GetTransaction, ListEntriesByTransaction, ListEntriesByAccount, GetAccountBalance exist and are used via the Wallet store
- [ ] Get Account Balance reads the derived balances view (or equivalent derivation from Entries); no writable/cached balance field on Account
- [ ] Integration test: retrieve a posted Ledger Transaction and all of its Entries
- [ ] Integration test: list Entries for an Account includes the expected lines after posting
- [ ] Integration test: Account Balance matches expected Amount after debit/credit posts given the Account’s Normal Balance (e.g. debit-normal vs credit-normal cases as needed)
- [ ] No speculative HTTP read APIs; store + tests only
