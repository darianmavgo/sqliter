# Unified Logging Plan

This plan outlines the steps to consolidate client-side (React) and server-side (Go) logging into a single stream, accessible via the server's standard output and log files.

## Goal
Enable real-time visibility of client-side events (errors, warnings, debug info) alongside server-side operations in the terminal running the `sqliter` server.

## Architecture

1.  **Server API**: Create a new endpoint `POST /sqliter/logs` that accepts log messages.
2.  **Client Transport**: Implement a mechanism in the React app to forward `console` logs to this endpoint.
3.  **Aggregation**: The server writes received client logs to its standard logging destination, properly tagged (e.g., `[CLIENT]`).

## Phase 1: Server Implementation

**Task**: Add a log ingestion endpoint.

-   **File**: `server/server.go`
-   **Endpoint**: `/sqliter/logs`
-   **Method**: `POST`
-   **Payload**:
    ```json
    {
      "level": "info",
      "message": "User clicked button",
      "context": {...}
    }
    ```
-   **Action**: Use standard Go `log` package to output: `[CLIENT] [INFO] User clicked button`.

## Phase 2: Client Implementation

**Task**: Intercept and forward logs.

-   **File**: `react-client/src/logger.js` (New File)
-   **Mechanism**:
    -   Override `console.log`, `console.error`, `console.warn`.
    -   Send `fetch` requests to `/sqliter/logs`.
    -   (Optional) Implement simple buffering/debouncing to prevent network flooding if logs are frequent.
-   **Integration**: Import this logger in `main.jsx` so it initializes immediately.

## Phase 3: "One Place" Access

-   **Terminal**: Running `./relaunch.sh` will now show mixed output:
    ```text
    [SERVER] Listening at: http://[::1]:8080
    [CLIENT] [INFO] App mounted
    [CLIENT] [DEBUG] Fetching /sqliter/fs
    [SERVER] Executing query: SELECT * FROM ...
    ```
-   **Log Files**: Since `config.hcl` specifies a `log_dir`, logs will also be persisted to disk in the `logs/` directory if the server uses a multi-writer (currently server uses `log.Printf` which goes to stderr/stdout unless configured otherwise, the existing `logError` logic writes to file, but standard logs might not. We can unify this too).

## Execution Steps

1.  **Modify Server**: Add `handleClientLogs` to `server.go` and register the route.
2.  **Create Client Logger**: Write `react-client/src/lib/logger.js` and import it in `main.jsx`.
3.  **Rebuild**: Run `./relaunch.sh` to build and start.
4.  **Verify**: Open the browser and watch the terminal.
