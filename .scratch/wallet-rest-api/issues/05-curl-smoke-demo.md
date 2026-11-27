# 05: Curl smoke demo for the five endpoints

**What to build:** A human can smoke-test the full wallet API against a running Wallet process with a curl script: create → get → balance → deposit → entries (and paging if useful), without a demo UI.

**Blocked by:** 04 (Page wallet Entries with keyset cursors).

**Status:** resolved

- [x] Script documents prerequisites (bootstrap/migrate, `make run-wallet`, base URL)
- [x] Script exercises all five endpoints end-to-end with sample Owner/Currency and a deposit Amount
- [x] Script shows how to follow next_cursor for a second entries page when present
- [x] No frontend or extra service binary is introduced

## Answer

Added `scripts/smoke-wallet.sh`: documents bootstrap/`make run-wallet`/BASE_URL, exercises create → get → balance → deposit → entries, and follows `next_cursor` after multiple deposits with `limit=2`. Verified live against `:8081`. No frontend/binary. No commit (per request).
