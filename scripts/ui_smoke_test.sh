#!/bin/bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://127.0.0.1:4173}"
PLAYWRIGHT_BROWSERS_PATH="${PLAYWRIGHT_BROWSERS_PATH:-/tmp/mockinterview-playwright-browsers}"

wait_for_http() {
  local url="$1"
  local label="$2"
  for _ in $(seq 1 30); do
    if curl -sf "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "ASSERT_FAIL[$label]: timed out waiting for $url" >&2
  exit 1
}

echo "STEP api health"
wait_for_http "$API_BASE_URL/api/health" "api_health"

echo "STEP playwright ui smoke"
(
  cd /Users/shiyi/mockinterview/web
  API_BASE_URL="$API_BASE_URL" \
  FRONTEND_URL="$FRONTEND_URL" \
  PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_PATH" \
  npm run test:e2e
)
