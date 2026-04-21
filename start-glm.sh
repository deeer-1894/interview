#!/bin/bash

# GLM-4.6V 快速启动脚本

echo "🚀 启动 Mock Interview 服务器 (GLM-4.6V)"
echo "=========================================="
echo ""
echo "模型配置:"
echo "  - Provider: openai-compatible"
echo "  - Model: glm-4.6v"
echo "  - Base URL: https://open.bigmodel.cn/api/paas/v4"
echo ""
echo "访问地址: http://localhost:8080"
echo ""
echo "按 Ctrl+C 停止服务器"
echo "=========================================="
echo ""

cd /Users/shiyi/mockinterview

export MODEL_PROVIDER="${MODEL_PROVIDER:-openai-compatible}"
export MODEL_NAME="${MODEL_NAME:-glm-4.6v}"
export MODEL_BASE_URL="${MODEL_BASE_URL:-https://open.bigmodel.cn/api/paas/v4}"
export MODEL_API_KEY="${MODEL_API_KEY:-${LLM_API_KEY:-}}"

if [ -z "$MODEL_API_KEY" ]; then
  echo "缺少 MODEL_API_KEY（或兼容变量 LLM_API_KEY），拒绝启动。" >&2
  exit 1
fi

go run ./cmd/mockinterview -serve -addr :8080
