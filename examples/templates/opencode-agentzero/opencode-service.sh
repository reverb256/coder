#!/bin/bash
set -euo pipefail

# OpenCode Agent Startup Script

CONFIG_PATH="${OPCODE_CONFIG_PATH:-/workspace/opencode-config.yaml}"
API_KEY="${OPENCODE_API_KEY:-}"

echo "[OpenCode] Starting with config: $CONFIG_PATH"
if [[ -z "$API_KEY" ]]; then
  echo "[OpenCode] ERROR: OPENCODE_API_KEY not set"
  exit 1
fi

# Start OpenCode service (replace with actual binary/command)
opencode-agent --config "$CONFIG_PATH" --api-key "$API_KEY" &

# Health check loop
for i in {1..10}; do
  if curl -sf http://localhost:8081/health; then
    echo "[OpenCode] Service healthy"
    break
  fi
  echo "[OpenCode] Waiting for service..."
  sleep 2
done

wait
