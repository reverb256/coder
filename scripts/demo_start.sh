#!/bin/bash
# Start mock OpenCode and Agent-Zero servers for demo

set -e

echo "Starting mock OpenCode server on :8081..."
nohup go run ./agentic/testdata/mock_opencode_server.go > /tmp/mock_opencode_server.log 2>&1 &

echo "Starting mock Agent-Zero server on :8082..."
nohup go run ./agentic/testdata/mock_agentzero_server.go > /tmp/mock_agentzero_server.log 2>&1 &

echo "Mock servers started."
echo "Logs: /tmp/mock_opencode_server.log, /tmp/mock_agentzero_server.log"
