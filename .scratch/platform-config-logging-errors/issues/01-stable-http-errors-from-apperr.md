# 01: Stable HTTP errors from shared app errors

**What to build:** A caller-facing HTTP error path where handlers can return a shared application error and the client receives a consistent status plus `{error:{code,message}}`, with unknown errors becoming a generic internal failure and no internals in the body.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

- [x] Shared application error type exists with stable codes (`invalid_argument`, `not_found`, `failed_precondition`, `conflict`, `internal`), client-safe message, and optional wrapped cause—no transport status on the type
- [x] HTTP helpers map those codes to 400 / 404 / 412 / 409 / 500 and write the agreed JSON error shape
- [x] Unknown errors map to `internal` / 500 with a generic client message; details stay out of the body
- [x] JSON success helper exists alongside error writing for consistent encoding
- [x] HTTP platform-seam tests cover mapped codes, unknown errors, and body shape via `httptest`

## Comments

- Implemented `internal/platform/apperr` + `internal/platform/httpx` (`WriteError` / `WriteJSON`). HTTP-seam tests green. Not committed (per request). Error-writer logging deferred to ticket 04.
