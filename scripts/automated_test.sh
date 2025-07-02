#!/bin/bash
# Master automation script for OpenCode + Agent-Zero integration testing

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_DIR="$ROOT_DIR/scripts"
REPORT_DIR="$ROOT_DIR/test_reports"
MOCK_LOG="$REPORT_DIR/mock_services.log"
CODER_LOG="$REPORT_DIR/coder.log"

mkdir -p "$REPORT_DIR"

echo "=== [1/8] Setting up environment..."
cp -v agentic/testdata/opencode_agentzero_mock.json "$REPORT_DIR/" || true
cp -v agentic/testdata/workspace_config.json "$REPORT_DIR/" || true
export AGENTIC_SECRETS_FILE="$REPORT_DIR/agentic_secrets.json"
export AGENTIC_SECRETS_PASSWORD="test-password"
export CODER_ENV="test"

echo "=== [2/8] Starting mock services..."
go run agentic/testdata/mock_opencode_server.go > "$MOCK_LOG" 2>&1 &
MOCK_OPENCODE_PID=$!
go run agentic/testdata/mock_agentzero_server.go >> "$MOCK_LOG" 2>&1 &
MOCK_AGENTZERO_PID=$!
sleep 3

echo "=== [3/8] Running all tests..."
bash "$SCRIPT_DIR/demo_test.sh"

echo "=== [4/8] Building Coder..."
go build -o "$REPORT_DIR/coder" ./cmd/coder
echo "=== [5/8] Starting Coder for verification..."
"$REPORT_DIR/coder" --help > "$CODER_LOG" 2>&1 || true
"$REPORT_DIR/coder" version >> "$CODER_LOG" 2>&1 || true

echo "=== [6/8] Collecting logs and reports..."
ls -lh "$REPORT_DIR"

echo "=== [7/8] Cleaning up..."
kill $MOCK_OPENCODE_PID $MOCK_AGENTZERO_PID || true
bash "$SCRIPT_DIR/demo_stop.sh" || true

echo "=== [8/8] Test automation complete ==="
if grep -q FAIL "$REPORT_DIR/unit_test.log"; then
  echo "[FAIL] Some tests failed. See $REPORT_DIR/unit_test.log"
  exit 1
else
  echo "[PASS] All tests passed. See $REPORT_DIR for details."
fi
