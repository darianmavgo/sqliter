# Multi-Target Deployment Plan for SQLiter

This document outlines the strategy to enable `sqliter` to serve as a versatile SQLite viewer and querying tool across three distinct deployment environments:
1.  **Independent Web Application** (Standalone binary / Docker)
2.  **Embedded Component in Flight3** (Library / Sub-router)
3.  **Desktop Application via Wails** (Native desktop wrapper)

## Core Philosophy: "Write Once, Run Everywhere"

To achieve this without maintaining three separate codebases, we will:
1.  **Libify the Backend**: Separate the "Business Logic" (Banquet parsing, Query composition, SQLite interaction) from the "Transport Layer" (HTTP Handlers).
2.  **Abstract the Frontend**: Create a Data Abstraction Layer (DAL) in the React client that can switch between `HTTP/REST` (for Web/Flight3) and `Wails/Events` (for Desktop).

---

## Phase 1: Backend Refactoring (The "Core")

We need to extract the logic currently buried in `sqliter/server.go` into reusable, transport-agnostic functions.

### 1.1 Create `sqliter.Core` / `sqliter.Engine`
Refactor logic from `handleAPI` and related HTTP handlers into a pure Go struct.

**Proposed Interface:**
```go
package sqliter

// Engine handles the core logic, agnostic of HTTP or Wails
type Engine struct {
    Config *Config
}

func NewEngine(cfg *Config) *Engine

// ListFiles returns a list of files in a directory (safe, strict relative paths)
func (e *Engine) ListFiles(dirRelPath string) ([]FileEntry, error)

// ListTables returns tables for a specific DB file
func (e *Engine) ListTables(dbRelPath string) ([]TableInfo, error)

// QueryOptions encapsulates sorting, filtering, limits
type QueryOptions struct {
    BanquetPath string
    Filter      string
    SortCol     string
    SortDir     string
    Offset      int
    Limit       int
}

// Query executes a query based on options and returns a specialized Result set
// generic enough for JSON marshaling OR Wails events.
func (e *Engine) Query(opts QueryOptions) (*QueryResult, error)
```

### 1.2 Update `sqliter.Server` (HTTP)
The `Server` struct will now be a thin wrapper around `Engine`.
- `NewServer(cfg)` initializes an `Engine`.
- `ServeHTTP` calls `Engine.ListFiles`, parses results, and writes JSON.
- **Benefit**: Keeps the HTTP logic minimal and standardizes error handling.

---

## Phase 2: Frontend Abstraction

The React client in `react-client/` currently calls `fetch('/sqliter/...')` directly. This won't work easily in Wails unless we run a local server (which is valid, but less "native"). Given the `wailssqliter` example uses events, we will support that pattern.

### 2.1 Interface Definition
Create `src/api/client.ts`:

```typescript
export interface DataClient {
    listFiles(dir: string): Promise<FileEntry[]>;
    listTables(db: string): Promise<TableInfo[]>;
    query(path: string, options: GridOptions): Promise<QueryResult>;
    streamQuery(path: string, options: GridOptions, onRow: (rows: any[])=>void): Promise<void>;
}
```

### 2.2 Implementations
1.  **`HttpClient`**: Implementation using `fetch`. Used for Standalone and Flight3.
2.  **`WailsClient`**: Implementation using `window.runtime` and `window.go` bindings. Connects to the methods exposed in `wailssqliter/app.go`.

### 2.3 Factory/Context
Use a React Context or Factory to select the client at startup.
- If `window.runtime` is detected (or a specific flag), use `WailsClient`.
- Otherwise, use `HttpClient`.

---

## Phase 3: Deployment Integration

### 3.1 Flight3 (Embedded)
**Goal**: `flight3` serves `sqliter` under `/sqliter/`.
**Configuration**:
- `flight3` imports `github.com/darianmavgo/sqliter`.
- In `flight3/cmd/serve.go` (or equivalent):
  ```go
  // Initialize SQLiter Server
  sqServer := sqliter.NewServer(&sqliter.Config{
      ServeFolder: app.DataDir(), // or specific restricted folder
      BaseURL:     "/sqliter",
  })

  // Mount
  router.Mount("/sqliter/", http.StripPrefix("/sqliter", sqServer))
  ```
- **Action**: Verify `flight3` dependencies and ensure `sqliter` version is updated.

### 3.2 Independent Web App
**Goal**: `sqliter serve ...`
**Configuration**:
- Remains as is, but functioning on top of the refactored `Engine`.
- The `go-build.sh` script continues to build the CLI tool.

### 3.3 Wails Desktop App
**Goal**: Native desktop experience.
**Configuration**:
- **Codebase**: `wailssqliter` (or a new `cmd/wails` inside `sqliter` repo?). *Suggestion: Keep `wailssqliter` as the desktop wrapper repo for clean separation usually, OR merge `cmd/desktop` into `sqliter` if desired. The user's prompt implies `wailssqliter` is separate, so we will update that.*
- **Backend**: `wailssqliter/app.go` imports `github.com/darianmavgo/sqliter`.
    - It maps `App.ExecuteQuery` -> `sqliter.Engine.Query`.
- **Frontend**:
    - `wails.json` points to `../sqliter/react-client`.
    - Build script: `cd ../sqliter/react-client && npm run build` -> Copy dist to `wails/frontend/dist` (or symlink).

---

## Action Plan Checklist

- [ ] **Step 1: Backend Extraction**
    - Refactor `sqliter/server.go` to extract `Engine` logic.
    - Ensure `Query` handles the complex Banquet parsing internally.
- [ ] **Step 2: Frontend Refactor**
    - Create `api/` layer in React client.
    - Implement `HttpClient`.
- [ ] **Step 3: Wails Protocol Match**
    - Define the exact `Event` names in Wails (e.g., `query_rows`, `query_done`) and implement the `WailsClient` in React to match.
- [ ] **Step 4: Integration Test**
    - Verify `sqliter serve` works (CLI).
    - Verify `flight3` (via Plan/Walkthrough).
    - Verify `wailssqliter` loads data.
