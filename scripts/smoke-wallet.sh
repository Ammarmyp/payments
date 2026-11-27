#!/usr/bin/env bash
# Smoke the five Wallet HTTP endpoints against a running process.
#
# Prerequisites:
#   1. Copy .env.example → .env (see docs/dev-bootstrap.md)
#   2. make bootstrap          # Postgres + Wallet migrations
#   3. make run-wallet         # listens on :8081 by default
#   4. curl + python3 on PATH (python used only to parse JSON)
#
# Usage:
#   ./scripts/smoke-wallet.sh
#   BASE_URL=http://localhost:8081 ./scripts/smoke-wallet.sh
#
# On Windows, run under Git Bash (not the WSL stub at System32\\bash.exe):
#   "C:\\Program Files\\Git\\bin\\bash.exe" scripts/smoke-wallet.sh
#
# Endpoints exercised:
#   POST /v1/wallets
#   GET  /v1/wallets/{id}
#   GET  /v1/wallets/{id}/balance
#   POST /v1/wallets/{id}/deposits
#   GET  /v1/wallets/{id}/entries  (including next_cursor paging)

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"
CURRENCY="${CURRENCY:-USD}"

json_field() {
  local field="$1"
  python -c 'import json,sys; print(json.load(sys.stdin)['"'"$field"'"'])'
}

echo "==> base URL: ${BASE_URL}"
echo "==> health"
curl -sS -f "${BASE_URL}/healthz"
echo

OWNER_ID="$(python -c 'import uuid; print(uuid.uuid4())')"
echo "==> POST /v1/wallets (owner=${OWNER_ID}, currency=${CURRENCY})"
CREATE_BODY="$(curl -sS -f -X POST "${BASE_URL}/v1/wallets" \
  -H 'Content-Type: application/json' \
  -d "{\"owner_id\":\"${OWNER_ID}\",\"currency\":\"${CURRENCY}\"}")"
echo "${CREATE_BODY}"
WALLET_ID="$(printf '%s' "${CREATE_BODY}" | json_field id)"

echo "==> GET /v1/wallets/${WALLET_ID}"
curl -sS -f "${BASE_URL}/v1/wallets/${WALLET_ID}"
echo

echo "==> GET /v1/wallets/${WALLET_ID}/balance (expect zero)"
curl -sS -f "${BASE_URL}/v1/wallets/${WALLET_ID}/balance"
echo

# Several deposits so a small page limit yields a next_cursor.
for amount in 100 200 300; do
  echo "==> POST /v1/wallets/${WALLET_ID}/deposits (amount=${amount})"
  curl -sS -f -X POST "${BASE_URL}/v1/wallets/${WALLET_ID}/deposits" \
    -H 'Content-Type: application/json' \
    -d "{\"amount\":${amount},\"currency\":\"${CURRENCY}\"}"
  echo
done

echo "==> GET /v1/wallets/${WALLET_ID}/balance (after deposits)"
curl -sS -f "${BASE_URL}/v1/wallets/${WALLET_ID}/balance"
echo

echo "==> GET /v1/wallets/${WALLET_ID}/entries?limit=2 (page 1)"
PAGE1="$(curl -sS -f "${BASE_URL}/v1/wallets/${WALLET_ID}/entries?limit=2")"
echo "${PAGE1}"
NEXT_CURSOR="$(printf '%s' "${PAGE1}" | python -c 'import json,sys; print(json.load(sys.stdin).get("next_cursor") or "")')"

if [[ -n "${NEXT_CURSOR}" ]]; then
  echo "==> GET /v1/wallets/${WALLET_ID}/entries?limit=2&cursor=... (page 2 via next_cursor)"
  curl -sS -f "${BASE_URL}/v1/wallets/${WALLET_ID}/entries?limit=2&cursor=${NEXT_CURSOR}"
  echo
else
  echo "==> next_cursor empty; skipping second page"
fi

echo "==> smoke ok"
