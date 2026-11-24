# 04: Panic recovery + Wallet HTTP stack wired

**What to build:** Panics on the Wallet HTTP stack become a controlled `internal` error response logged at error with the request ID; the Wallet mux runs request ID outside recovery; existing health and readiness keep working; error writing logs client-mapped failures at info and internals/panics at error.

**Blocked by:** 01 (Stable HTTP errors from shared app errors), 02 (Request ID on responses, context, and logs), 03 (Collect-all log config at process boot).

**Status:** ready-for-human

- [x] Panic recovery middleware logs the panic at error (with request ID when present) and responds as `internal` / 500 with the shared error JSON shape
- [x] Wallet process wraps its mux with request ID outside recovery; listen address stays hardcoded as today
- [x] `/healthz` and `/readyz` still behave as before under that stack
- [x] Error writer uses info for mapped client application errors and error for `internal`/unknown/panic paths
- [x] HTTP platform-seam tests cover panic → controlled response and log level behaviour; manual smoke still passes health/ready with `X-Request-ID` present

## Comments

- Implemented `httpx.Recover`, WriteError log levels, Wallet `RequestID(Recover(...))` at `:8081`. Health tests run under that stack and assert `X-Request-ID`. Seam tests green. Live process smoke not run here (Postgres may be down). Not committed (per request).
