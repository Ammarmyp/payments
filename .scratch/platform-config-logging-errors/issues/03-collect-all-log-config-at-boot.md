# 03: Collect-all log config at process boot

**What to build:** The Wallet process reads `LOG_LEVEL` and `LOG_FORMAT` through a shared env helper that reports every problem at once, builds its structured logger from that config (service name/version from process constants), and exits before listen when configuration is invalid; example env and bootstrap docs list the new variables.

**Blocked by:** 02 (Request ID on responses, context, and logs).

**Status:** ready-for-human

- [x] Shared env helper reads typed values with defaults and returns one error listing every problem (no fail-fast on the first bad key)
- [x] Wallet boot uses the helper for `LOG_LEVEL` (default `info`) and `LOG_FORMAT` (`text`|`json`, default `text`) and constructs the shared logger before serving
- [x] Invalid or contradictory log settings cause a non-zero exit before the server listens, with all problems visible
- [x] Postgres pool env loading remains unchanged
- [x] Boot-config seam tests cover defaults and collect-all failures
- [x] `.env.example` and local bootstrap docs document `LOG_LEVEL` and `LOG_FORMAT`

## Comments

- Implemented `internal/platform/envconfig` + Wallet boot wiring + docs. Seam tests green. Not committed (per request). Request ID / recovery wrapping remains ticket 04.
