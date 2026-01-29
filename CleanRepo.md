# Repository Cleanup & Build Guide

This document summarizes the current state of the `sqliter` repository after the migration to a React + AG Grid client.

## Build Requirements

### 1. React Client
The frontend is a React application located in `react-client/`.
- **Location**: `react-client/`
- **Dependencies**: Node.js, `npm`.
- **Build Command**: `npm run build` (inside `react-client/`)
- **Output**: `react-client/dist/` (contains static assets: index.html, js, css).

### 2. Go Server
The backend is a Go server that embeds the built React assets.
- **Dependencies**: Go 1.24+.
- **Pre-requisite**: The React app **MUST** be built first and the `dist` folder copied to `server/ui`.
- **Embed Path**: `server/ui` (The `server/server.go` file embeds `ui/*`).
- **Build Command**: 
  ```bash
  # 1. Build Client
  cd react-client && npm run build
  # 2. Update Server Assets
  rm -rf ../server/ui && mkdir -p ../server/ui && cp -R dist/* ../server/ui/
  # 3. Build Server
  cd .. && go build ./cmd/sqliter
  ```

## Configuration (`config.hcl`)

The `config.hcl` file is still used by the Go server.
**Required Fields**:
- `data_dir`: Directory containing SQLite files to serve.
- `port`: Port to listen on (e.g., "8080").
- `enable_wasm`: (Optional) If true, enables WASM/Canvas viewer features (currently distinct from the main React/AG Grid view).

**Obsolete/Unused Fields** (Can likely be removed from usage, though `Config` struct might still have them):
- `template_dir`: We no longer use `html/template` for the main UI.
- `stylesheet`: We render via React/Vite CSS, not injected stylesheets.
- `sticky_header`: Handled by AG Grid.
- `auto_redirect_single_table`: Logic exists in API but might not be relevant for SPA flow (SPA handles routing).
- `auto_select_tb0`: API logic still respects this for query generation.

## Helpful Scripts

- **`go_build.sh`**: Should be updated to include the React build and asset copy steps to ensure a single command builds the full app.
- **`relaunch.sh`**: Useful for development loop (kill implementation TBD by user), but needs to ensure it triggers the React build if frontend changes occurred.

## Cleanup Recommendations (Safe to Delete)

The following files and directories are likely obsolete following the complete migration to React/AG Grid:

### Code
- **`sqliter/html_table.go`**: *Already Deleted*.
- **`sqliter/fallback.go`**: *Already Deleted*.
- **`sqliter/templates/`**:
    - `head.html`, `foot.html`, `row.html`: **DELETE**. These were for server-side HTML rendering.
    - `default.css`, `default.js`: **DELETE**. We use `react-client/src/index.css` and React components now.
    - `wasm_table.html`: **KEEP** if you still support the specific `/sqliter/wasm` flow, otherwise **DELETE**.
    - `wasm_exec.js`: **KEEP** if WASM is still relevant.

### Tests
- **`tests/table_writer_test.go`**: *Already Deleted?* If not, it tests the old `TableWriter`. **DELETE**.
- **`tests/html_strictness_test.go`**: *Already Deleted*.
- **`tests/ux_structure_test.go`**: *Already Deleted*.
- **`tests/links_browser_test.go`**: *Already Deleted*.
- **`cmd/demomemory`**: *Already Deleted*.

### Other
- **`cmd/servelocal`**: Check if this uses `html_table`. If so, it's broken or needs update. If unused, **DELETE**.
