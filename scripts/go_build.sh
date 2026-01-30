#!/bin/bash
set -e

# 1. Build React Client
echo "Building React Client..."
cd react-client
npm install
npm run build
cd ..

# 2. Update Server Assets
echo "Updating Server Assets..."
rm -rf server/ui
mkdir -p server/ui
cp -R react-client/dist/* server/ui/

# 3. Build Go Server
echo "Building Go Server..."
mkdir -p bin
CGO_ENABLED=0 go build -o bin/ ./cmd/sqliter
echo "Build Complete. Binary is in bin/sqliter"
