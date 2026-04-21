#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
MONGO_DATABASE="${MONGO_DATABASE:-mockinterview}"
MODEL_PROVIDER="${MODEL_PROVIDER:-openai-compatible}"
MODEL_NAME="${MODEL_NAME:-glm-4.6v}"
MODEL_API_KEY="${MODEL_API_KEY:-${LLM_API_KEY:-6030658fa75b4499ae1937a52f9533e5.gOON1QaKhKggSiAR}}"
MODEL_BASE_URL="${MODEL_BASE_URL:-https://open.bigmodel.cn/api/paas/v4}"
MODEL_TIMEOUT_SECONDS="${MODEL_TIMEOUT_SECONDS:-180}"
ADDR="${ADDR:-:8080}"

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
echo "  - API Key: 使用脚本默认值，可被环境变量覆盖"
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
