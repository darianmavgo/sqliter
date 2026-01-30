#!/bin/bash
set -e

# 1. Kill old processes
echo "Killing old instances..."
pkill -f aggrid-backend || true

# 2. Build React Client
echo "Building React client..."
cd react-client
npm install
npm run build
cd ..

# 3. Build & Run Go Server
echo "Building and starting Go server..."
CGO_ENABLED=0 go build -o bin/sqliter ./cmd/sqliter

# Start in background
./bin/sqliter sample_data > server.log 2>&1 &
SERVER_PID=$!

echo "Server running with PID $SERVER_PID"
echo "Waiting for server to initialize..."

# 4. Wait for URL and Open Chrome
url=""
count=0
while [ -z "$url" ] && [ $count -lt 30 ]; do
  if grep -q "SERVING_AT=" server.log; then
    url=$(grep "SERVING_AT=" server.log | cut -d= -f2)
  fi
  sleep 0.5
  count=$((count+1))
done

if [ -z "$url" ]; then
  echo "Failed to get server URL from logs."
  cat server.log
  kill $SERVER_PID
  exit 1
fi

echo "App running at: $url"
echo "Opening Chrome..."

# Try to open Google Chrome specifically if available on Mac, else default open
if [ -d "/Applications/Google Chrome.app" ]; then
    open -a "Google Chrome" "$url"
else
    open "$url"
fi

# Keep script running to maintain the background process
wait $SERVER_PID
