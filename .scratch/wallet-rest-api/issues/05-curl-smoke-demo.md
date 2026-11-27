# 05: Curl smoke demo for the five endpoints

**What to build:** A human can smoke-test the full wallet API against a running Wallet process with a curl script: create → get → balance → deposit → entries (and paging if useful), without a demo UI.

**Blocked by:** 04 (Page wallet Entries with keyset cursors).

**Status:** ready-for-agent

- [ ] Script documents prerequisites (bootstrap/migrate, `make run-wallet`, base URL)
- [ ] Script exercises all five endpoints end-to-end with sample Owner/Currency and a deposit Amount
- [ ] Script shows how to follow next_cursor for a second entries page when present
- [ ] No frontend or extra service binary is introduced
