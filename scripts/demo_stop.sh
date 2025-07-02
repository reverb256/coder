#!/bin/bash
# Stop mock OpenCode and Agent-Zero servers for demo

set -e

echo "Stopping mock OpenCode and Agent-Zero servers..."

pkill -f 'go run ./agentic/testdata/mock_opencode_server.go' || true
pkill -f 'go run ./agentic/testdata/mock_agentzero_server.go' || true

echo "Mock servers stopped."
