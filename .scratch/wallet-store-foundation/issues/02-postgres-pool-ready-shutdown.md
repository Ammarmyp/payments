# 02: Wallet process reaches Postgres (pool, ready, shutdown)

**What to build:** The Wallet process opens one configurable PostgreSQL pool with a bounded startup ping, exposes liveness without the database and readiness with it, and shuts down in order: mark not ready → drain HTTP → close the pool.

**Blocked by:** None (can start immediately; parallel with 01).

**Status:** ready-for-human

- [x] Shared platform postgres pool helper exists with conservative, optionally overridable pool settings
- [x] Required database URL is read at startup; initial ping uses a bounded timeout; failure fails startup cleanly without leaking the pool
- [x] `GET /healthz` succeeds without requiring PostgreSQL
- [x] `GET /readyz` returns 200 when PostgreSQL is reachable and 503 when it is not or the process is shutting down; readiness ping uses a bounded timeout
- [x] Graceful shutdown order is: not-ready → stop accepting new work / drain HTTP with a deadline → close pool → exit
- [x] HTTP tests cover health without DB and ready success/failure paths with injectable pool and readiness gate

## Answer

Added `internal/platform/postgres` (`ConfigFromEnv` + `Open` with startup ping and close-on-failure) and `internal/wallet` health/ready handlers with injectable `Pinger` + `ReadyGate`. Wired `cmd/wallet` to open the pool, serve `/healthz`/`/readyz`, and shut down not-ready → HTTP drain → pool close. HTTP tests cover liveness without DB, ready success/failure/shutdown/timeout, and response bodies. No commit (manual review requested).
