#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo ">>> Building frontend..."
cd web && npm run build
cd "$SCRIPT_DIR"
rm -rf internal/api/frontend
cp -r web/build internal/api/frontend

echo ">>> Building backend..."
go build -o bin/timbre-server ./cmd/server

echo ">>> Starting timbre-server..."
MS_DATA_DIR=./data ./bin/timbre-server
