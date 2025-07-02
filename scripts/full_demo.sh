#!/bin/bash
# Full end-to-end demo: runs all tests, starts Coder, creates demo workspace, executes workflows

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_DIR="$ROOT_DIR/scripts"
REPORT_DIR="$ROOT_DIR/test_reports"
DEMO_LOG="$REPORT_DIR/full_demo.log"
CODER_BIN="$REPORT_DIR/coder"

mkdir -p "$REPORT_DIR"

echo "=== [1/6] Running complete automated test suite ==="
bash "$SCRIPT_DIR/automated_test.sh" | tee "$DEMO_LOG"

echo "=== [2/6] Starting Coder instance for demo ==="
"$CODER_BIN" serve --dev > "$REPORT_DIR/coder_demo.log" 2>&1 &
CODER_PID=$!
sleep 5

echo "=== [3/6] Creating demo workspace with OpenCode/Agent-Zero ==="
curl -sf -X POST http://localhost:8080/api/tasks -d '{"type":"demo","payload":{"workflow":"sample"}}' -H "Content-Type: application/json" | tee -a "$DEMO_LOG"

echo "=== [4/6] Executing sample workflows ==="
curl -sf http://localhost:8080/api/tasks/demo | tee -a "$DEMO_LOG"

echo "=== [5/6] UI validation (headless) ==="
curl -sf http://localhost:8080/health | tee -a "$DEMO_LOG"

echo "=== [6/6] Cleaning up demo ==="
kill $CODER_PID || true
bash "$SCRIPT_DIR/demo_stop.sh" || true

echo "=== Demo complete. See $REPORT_DIR/full_demo.log for details. ==="
