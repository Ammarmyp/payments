# Local bootstrap

How to bring the stack up after opening the IDE (or after a reboot), then develop and test.

## First time only

1. Copy env file:

   ```powershell
   cp .env.example .env
   ```

2. Confirm Docker Desktop is running.

3. Install CLI tools if missing: `docker`, `migrate` ([golang-migrate](https://github.com/golang-migrate/migrate)), `sqlc`, Go.

## Every reopen

```powershell
make bootstrap
```

That starts Postgres/Redis/NATS (`docker compose up -d --wait`) and applies Wallet migrations from `DATABASE_URL` in `.env`.

Then in separate terminals:

```powershell
make run-wallet   # listens on :8081 — needs DATABASE_URL
make test         # needs TEST_DATABASE_URL
```

Smoke the process: `GET http://localhost:8081/healthz` and `GET http://localhost:8081/readyz`.

## Make targets

| Target | What it does |
|--------|----------------|
| `make bootstrap` | `up` + `migrate-up` |
| `make up` | Start compose services and wait for health |
| `make down` | Stop compose services |
| `make migrate-up` | Apply migrations for `SERVICE` (default `wallet`) |
| `make migrate-down` | Roll back one migration |
| `make migrate-version` | Print migration version |
| `make migrate-new name=...` | Create a new migration pair |
| `make sqlc` | Regenerate Wallet sqlc from migrations + queries |
| `make run-wallet` | `go run ./cmd/wallet` |
| `make test` | `go test ./...` with `TEST_DATABASE_URL` |

## Env vars (`.env`)

| Variable | Used by |
|----------|---------|
| `POSTGRES_*`, `REDIS_PASSWORD` | Docker Compose |
| `DATABASE_URL` | Wallet process, Wallet migrations |
| `TEST_DATABASE_URL` | Integration tests |

Do not put real secrets in `.env.example`. `.env` is gitignored.

## Notes

- Init script `deploy/postgres/01-create-database.sql` creates `wallet`, `auth`, and `transfer` databases on first volume create only.
- Redis and NATS start with compose but are not required for Wallet foundation work yet.
- Other services: `make migrate-up SERVICE=auth` (builds DB URL from `POSTGRES_USER` / `POSTGRES_PASSWORD`).
- After changing Wallet migrations or sqlc query files: `make sqlc` (sqlc validates queries against `db/wallet/migrations` `*.up.sql`).
- New migrations: use `make migrate-new name=...` so names stay zero-padded and sqlc order matches golang-migrate.
