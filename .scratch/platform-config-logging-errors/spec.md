# Platform config, logging, and errors

Status: ready-for-agent

## Problem Statement

Services in this payments monorepo have no shared way to read process configuration, emit structured logs with request correlation, or turn application failures into consistent HTTP responses. The Wallet process today builds a bare logger inline, hardcodes listen behaviour without collect-all env validation for log settings, and has no request ID or shared error contract—so the next HTTP surfaces would each invent their own plumbing.

## Solution

Introduce shared platform packages for application errors, typed environment configuration, structured logging, and HTTP helpers (request ID, JSON, error writing, panic recovery). Wire them into the Wallet process so every request gets a request ID, logs carry service identity and that ID, log level/format come from the environment with collect-all misconfig reporting, and app errors map to one stable JSON error body—without changing Wallet ledger/store behaviour or Postgres pool env loading.

## User Stories

1. As a platform operator, I want log verbosity controlled by an environment variable with a safe default, so that I can debug without rebuilding.
2. As a platform operator, I want log encoding selectable as text or JSON via an environment variable, so that local terminals stay readable and collectors get structured lines.
3. As a platform operator, I want every configuration problem reported in one startup failure, so that I fix all missing or invalid settings in a single pass.
4. As a platform operator, I want the process to refuse to listen when configuration is invalid, so that a half-configured instance never serves traffic.
5. As a platform operator, I want startup and runtime logs to name the service and version, so that aggregated logs are attributable to a binary.
6. As a platform operator, I want local text logs to show a short time of day, so that terminal output stays scannable.
7. As a platform operator, I want JSON logs when configured for collectors, so that downstream parsing does not depend on text layout.
8. As an API caller, I want every response to include a request ID header, so that I can correlate my call with server logs.
9. As an API caller, I want my valid request ID honored when I send one, so that one ID can follow a call across services.
10. As an API caller, I want the server to mint a request ID when I omit one or send an invalid one, so that correlation still works.
11. As an API caller, I want error responses to use one JSON shape with a stable code and a safe message, so that clients can branch on code without scraping prose.
12. As an API caller, I want error bodies free of stack traces and internal wrap chains, so that implementation details are not leaked.
13. As an API caller, I want unknown failures to surface as an internal error with a generic client message, so that unexpected bugs do not expose internals.
14. As an API caller, I want panics to become a controlled internal error response, so that a bug does not drop the connection without a usable body.
15. As an on-call engineer, I want the request ID on every log line for a request, so that I can grep one incident end to end.
16. As an on-call engineer, I want client-mapped failures logged at info and true internals/panics at error, so that alert noise stays tied to outages.
17. As a service developer, I want a shared application error type that describes what went wrong without an HTTP status, so that HTTP and later gRPC can map the same failure differently.
18. As a service developer, I want a small fixed set of error codes for this slice, so that mapping tables stay reviewable.
19. As a service developer, I want JSON write helpers next to error writing, so that success and failure responses stay consistent.
20. As a service developer, I want context-scoped log attributes rather than a logger stored in context, so that one process logger stays the source of truth.
21. As a service developer, I want Wallet domain store sentinels left unchanged, so that ledger code keeps expressing domain failures with `errors.Is`.
22. As a service developer, I want Postgres pool environment loading left where it is, so that this slice does not reshape database bootstrap.
23. As a Wallet operator, I want existing health and readiness endpoints to keep working under the new middleware, so that orchestrators do not break.
24. As a Wallet operator, I want the listen address to remain the current hardcoded port for now, so that runbooks and smoke checks stay valid.
25. As a future service owner, I want these packages under shared platform modules, so that Auth or Transfer can reuse the same patterns.
26. As a test engineer, I want HTTP-level tests of request ID, error bodies, panic recovery, and log correlation, so that the stack is locked at the edge.
27. As a test engineer, I want env-collector tests that prove multiple problems surface together, so that collect-all behaviour cannot regress.
28. As a documentation reader, I want new log-related env vars listed beside existing ones, so that local setup stays copy-pasteable.
29. As an agent implementing later work, I want this effort tracked under a scratch feature with a spec and tickets, so that AFK runs have a clear frontier.
30. As a domain collaborator, I want glossary Wallet terms (Account, Ledger Transaction, and the rest) unused as names for these platform concepts, so that platform plumbing is not mistaken for ledger language.

## Implementation Decisions

- Add shared platform modules for: application errors; typed environment collection; structured logging; HTTP middleware and JSON/error helpers.
- Application errors carry a stable string code, a client-safe message, and an optional wrapped cause; they do not carry HTTP or gRPC status codes.
- Initial codes: `invalid_argument`, `not_found`, `failed_precondition`, `conflict`, `internal`. HTTP mapping: 400, 404, 412, 409, 500 respectively.
- HTTP error JSON contract: top-level `error` object with `code` and `message` fields only for this slice.
- Unknown errors and panics map to `internal` with a generic client message; details go to logs only.
- Environment helper reads typed values and accumulates every problem into one returned error; no third-party env library.
- Process env for this slice: `LOG_LEVEL` (default `info`) and `LOG_FORMAT` (`text` or `json`, default `text`). Invalid values are collect-all failures.
- Service name and version are constants supplied by the process entrypoint into the logger constructor, not operator env vars.
- Logger is built with `log/slog`; text handler uses a short clock time for terminals; JSON handler is selected via `LOG_FORMAT`.
- Context holds log attributes only (including request ID); call sites use the process logger’s context-aware methods.
- Request ID header name: `X-Request-ID`. Accept inbound values up to 64 characters of letters, digits, dash, underscore; otherwise mint a new ID (UUID). Put the ID in context, on log lines, and on the response header.
- Middleware composition: request ID wrapper outside panic recovery so recovery logs see the request ID. Include JSON helpers, error writer, request ID middleware, and recovery. No access-log middleware in this slice.
- Error writer logs mapped client codes at info and `internal`/panics at error, always with request ID and code attributes when available.
- Wire the Wallet process entrypoint to: collect log config; exit before listen on failure; build the shared logger; wrap the existing mux with request ID and recovery. Keep listen address hardcoded as today.
- Do not migrate Postgres pool env into the new config helper in this slice; do not add a Wallet store → application-error mapper yet.
- No new ADR for this slice; package docs and this scratch feature carry the rationale.
- Update example env and local bootstrap docs for the new log variables.
- Track work under the scratch feature directory for this effort; implementation issues are separate ticket files.

## Testing Decisions

- Good tests assert external behaviour (HTTP status/headers/body; env collect-all outcomes; log line attributes/levels visible to a test writer), not private helpers or handler implementation structure.
- Seams:
  1. **HTTP platform stack (primary):** exercise request ID, recovery, and error writing through the shared HTTP helpers with `httptest`, capturing logs in a buffer. Assert status, JSON error shape, `X-Request-ID` honor/mint behaviour, and correlation/level on log output.
  2. **Boot config (secondary):** exercise the env collector’s public API with in-process environment values. Assert defaults and that multiple invalid settings appear in one error.
- Do not add a third seam that execs the Wallet binary; do not unit-test unexported helpers except as behaviour visible through the seams above.
- No Wallet store or real PostgreSQL required for these tests; Postgres pool and ledger tests remain untouched.
- Prior art: Wallet health handler tests using `httptest`; platform postgres tests using an external test package and skip-on-missing-DB (not required here).

## Out of Scope

- Mapping Wallet store sentinel errors into application errors.
- Moving Postgres/`DATABASE_*` loading into the shared env helper.
- Making the HTTP listen address configurable.
- Access/request logging middleware.
- gRPC (or other non-HTTP) status mapping (design must not preclude it later).
- Field-level validation error payloads beyond code + message.
- Changing `/healthz` or `/readyz` semantics beyond running them under the new middleware.
- Auth, Transfer, or other service binaries beyond reusable packages and Wallet wiring.
- Glossary or ADR additions for this platform work.

## Further Notes

- This work is platform engineering vocabulary; keep root glossary language reserved for the Wallet bounded context.
- After implementation: run the usual test target, smoke Wallet health/ready, and confirm a request shows `X-Request-ID` and that bad `LOG_LEVEL`/`LOG_FORMAT` combinations print all problems and exit before listen.
- Tracer-bullet tickets under this feature should stay vertical: each slice demoable at the HTTP or boot seam, with wiring tickets blocked by the packages they need.
