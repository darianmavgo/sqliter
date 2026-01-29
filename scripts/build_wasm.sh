#!/bin/bash
set -e

echo "Building WASM module for SQLiter canvas renderer..."

# Build WASM binary
GOOS=js GOARCH=wasm go build -o bin/sqliter.wasm cmd/wasmviewer/main.go

# Copy wasm_exec.js from Go runtime
GOROOT=$(go env GOROOT)
WASM_EXEC_PATH="$GOROOT/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC_PATH" ]; then
    # Fallback to misc/wasm location for older Go versions
    WASM_EXEC_PATH="$GOROOT/misc/wasm/wasm_exec.js"
fi
cp "$WASM_EXEC_PATH" sqliter/templates/

# Get file sizes
WASM_SIZE=$(du -h bin/sqliter.wasm | cut -f1)
GZIP_SIZE=$(gzip -c bin/sqliter.wasm | wc -c | awk '{printf "%.1fM", $1/1024/1024}')

echo "✓ WASM build complete:"
echo "  - Binary: bin/sqliter.wasm ($WASM_SIZE)"
echo "  - Gzipped: ~$GZIP_SIZE"
echo "  - Runtime: sqliter/templates/wasm_exec.js"
