# 02: Request ID on responses, context, and logs

**What to build:** Every HTTP request gets an `X-Request-ID` (honor a valid inbound value, otherwise mint one), that ID is available on the request context, and structured logs from a shared logger include service name, version, and context attributes so `request_id` appears on lines for that request.

**Blocked by:** None (can start immediately; parallel with 01).

**Status:** ready-for-human

- [x] Structured logger constructor writes to a given writer with service name and version on every line; text uses short time-of-day; JSON encoding is available as a format option
- [x] Context can carry log attributes (not a logger); context-aware log methods merge those attributes into records
- [x] Log level parsing accepts debug/info/warn/error in any common letter case
- [x] Request ID middleware honors valid inbound `X-Request-ID` (≤64 letters, digits, dash, underscore), mints otherwise, sets response header, and stores the ID as a context log attribute
- [x] HTTP platform-seam tests with a log buffer prove honor/mint behaviour and `request_id` on log output

## Comments

- Implemented `internal/platform/logging` + `httpx.RequestID`. Seam tests green. Not committed (per request). Env collect-all and Wallet wiring remain tickets 03–04.
