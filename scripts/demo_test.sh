#!/bin/bash
# Enhanced: Run all tests, benchmarks, API checks, and generate reports for the demo environment

set -euo pipefail

REPORT_DIR="test_reports"
COVERAGE_FILE="$REPORT_DIR/coverage.out"
COVERAGE_HTML="$REPORT_DIR/coverage.html"
BENCH_FILE="$REPORT_DIR/bench.txt"
API_LOG="$REPORT_DIR/api_checks.log"
MEMCHECK_FILE="$REPORT_DIR/memcheck.txt"

mkdir -p "$REPORT_DIR"

echo "=== [1/7] Running unit tests with coverage..."
go test -v -coverprofile="$COVERAGE_FILE" ./agentic/... | tee "$REPORT_DIR/unit_test.log"
go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"

echo "=== [2/7] Running integration demo..."
# Ensure port 8080 is free before starting the integration demo
fuser -k 8080/tcp || true
go run ./agentic/cmd/example/main.go > "$REPORT_DIR/integration_demo.log" 2>&1 &

SERVER_PID=$!
sleep 3

echo "=== [3/7] API endpoint checks..." | tee "$API_LOG"
curl -sf http://localhost:8080/health | tee -a "$API_LOG"
curl -sf -X GET http://localhost:8080/api/user -H "Authorization: Bearer test" | tee -a "$API_LOG" || echo "User endpoint requires auth" >> "$API_LOG"

echo "=== [4/7] Performance benchmarks..."
go test -bench=. ./agentic/... | tee "$BENCH_FILE"

echo "=== [5/7] Memory race checks..."
go test -race ./agentic/... | tee "$MEMCHECK_FILE"

echo "=== [6/7] Stopping integration server..."
kill $SERVER_PID || true

echo "=== [7/7] Test summary ==="
echo "Unit test log: $REPORT_DIR/unit_test.log"
echo "Coverage HTML: $COVERAGE_HTML"
echo "Benchmarks: $BENCH_FILE"
echo "API checks: $API_LOG"
echo "Memory/race: $MEMCHECK_FILE"
echo "Integration demo log: $REPORT_DIR/integration_demo.log"

if grep -q FAIL "$REPORT_DIR/unit_test.log" || grep -q FAIL "$BENCH_FILE" || grep -q FAIL "$MEMCHECK_FILE"; then
  echo "[FAIL] Some tests failed. See logs above."
  exit 1
else
  echo "[PASS] All tests passed."
fi
