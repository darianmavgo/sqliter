#!/bin/bash
set -e

# Build the React client
echo "Building React client..."
cd react-client
npm install
npm run build
cd ..

# Run the Go server which serves the client
echo "Starting Go server..."
cd go-server
# We use a subshell or background process to capture output, but simple 'go run' is fine.
# The server prints "SERVING_AT=..."
# We want to capture that and open it.

# Kill any existing on default port just in case (optional, user manages kill usually)

# Compile first to ensure speed
go build -o server main.go

# Run server in background, capture output to file to find the port
./server > server.log 2>&1 &
SERVER_PID=$!

echo "Server running with PID $SERVER_PID"
echo "Waiting for server to initialize..."

# Wait for the "SERVING_AT" line
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
echo "Opening browser..."
open "$url"

# Keep script running to keep server alive
wait $SERVER_PID
