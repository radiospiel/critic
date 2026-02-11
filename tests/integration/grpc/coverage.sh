#!/usr/bin/env bash
#
# Generate a coverage report for the gRPC integration tests.
#
# Outputs:
#   coverage_grpc.out  – raw Go coverage profile
#   coverage_grpc.html – HTML coverage report (open in browser)
#   per-function summary printed to stdout
#
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Ensure the embedded dist directory exists (needed for compilation)
mkdir -p src/webui/dist
touch src/webui/dist/.gitkeep

echo "=== Running gRPC integration tests with coverage ==="
go test ./tests/integration/grpc/... \
    -v \
    -coverprofile=coverage_grpc.out \
    -coverpkg=./src/api/server/...

echo ""
echo "=== Per-function coverage ==="
go tool cover -func=coverage_grpc.out

echo ""
echo "=== Generating HTML report ==="
go tool cover -html=coverage_grpc.out -o coverage_grpc.html
echo "HTML report written to coverage_grpc.html"
