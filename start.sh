#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
MONGO_DATABASE="${MONGO_DATABASE:-mockinterview}"
MODEL_PROVIDER="${MODEL_PROVIDER:-}"
MODEL_NAME="${MODEL_NAME:-}"
MODEL_API_KEY="${MODEL_API_KEY:-${LLM_API_KEY:-}}"
MODEL_BASE_URL="${MODEL_BASE_URL:-}"
MODEL_TIMEOUT_SECONDS="${MODEL_TIMEOUT_SECONDS:-180}"
ADDR="${ADDR:-:8080}"

missing_vars=()
[[ -n "$MODEL_PROVIDER" ]] || missing_vars+=("MODEL_PROVIDER")
[[ -n "$MODEL_NAME" ]] || missing_vars+=("MODEL_NAME")
[[ -n "$MODEL_API_KEY" ]] || missing_vars+=("MODEL_API_KEY or LLM_API_KEY")
[[ -n "$MODEL_BASE_URL" ]] || missing_vars+=("MODEL_BASE_URL")

if [[ ${#missing_vars[@]} -gt 0 ]]; then
  echo "缺少模型环境变量，拒绝启动: ${missing_vars[*]}" >&2
  exit 1
fi

echo "启动 Mock Interview 服务器"
echo "================================"
echo ""
echo "持久化模式: Redis + Mongo"
echo "  REDIS_ADDR=$REDIS_ADDR"
echo "  MONGO_URI=$MONGO_URI"
echo "  MONGO_DATABASE=$MONGO_DATABASE"
echo ""
echo "模型配置:"
echo "  - Provider: $MODEL_PROVIDER"
echo "  - Model: $MODEL_NAME"
echo "  - Base URL: $MODEL_BASE_URL"
echo "  - Timeout: ${MODEL_TIMEOUT_SECONDS}s"
echo "  - API Key: 通过环境变量注入"
echo ""
DISPLAY_ADDR="${ADDR#:}"
if [[ "$DISPLAY_ADDR" != :* ]]; then
  DISPLAY_ADDR=":$DISPLAY_ADDR"
fi
echo "访问地址: http://localhost$DISPLAY_ADDR"
echo "按 Ctrl+C 停止服务器"
echo "================================"
echo ""

export REDIS_ADDR
export MONGO_URI
export MONGO_DATABASE
export MODEL_PROVIDER
export MODEL_NAME
export MODEL_API_KEY
export MODEL_BASE_URL
export MODEL_TIMEOUT_SECONDS

go run ./cmd/mockinterview -serve -addr "$ADDR"
