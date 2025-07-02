#!/bin/bash
set -euo pipefail

# Agent-Zero Agent Startup Script

CONFIG_PATH="${AGENTZERO_CONFIG_PATH:-/workspace/agentzero-config.yaml}"
API_KEY="${AGENTZERO_API_KEY:-}"

echo "[Agent-Zero] Starting with config: $CONFIG_PATH"
if [[ -z "$API_KEY" ]]; then
  echo "[Agent-Zero] ERROR: AGENTZERO_API_KEY not set"
  exit 1
fi

# Start Agent-Zero service (replace with actual binary/command)
agentzero-agent --config "$CONFIG_PATH" --api-key "$API_KEY" &

# Health check loop
for i in {1..10}; do
  if curl -sf http://localhost:8090/health; then
    echo "[Agent-Zero] Service healthy"
    break
  fi
  echo "[Agent-Zero] Waiting for service..."
  sleep 2
done

wait
