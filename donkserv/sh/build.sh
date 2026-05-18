#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "Building donk..."
go build -ldflags="-s -w" -o sh/donk ./cmd/...

echo "Build successful: sh/donk"
