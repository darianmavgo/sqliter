# Adopt React AG Grid Plan

This plan outlines the steps to replace the existing server-side rendered HTML tables (using `html_table.go`) with a Modern React client using AG Grid.

## Goal
Completely replace the `html_table.go` templating system with the `react-client` application, served by the Go backend. The Go backend will transition from serving HTML to serving the React SPA and providing a JSON API.

## Phase 1: Go Server API Implementation
The server needs to provide data in JSON format instead of HTML.

1.  **Create API Endpoints**:
    - Modify `server/server.go` to support a mode (content negotiation or dedicated `/api` prefix) where it returns JSON.
    - **Endpoint**: `/rows` (or `/api/rows`)
        - **Parameters**: `db` (database path), `table` (table name), `start` (offset), `end` (limit), `sortCol`, `sortDir`.
        - **Response**:
          ```json
          {
            "columns": ["id", "name", ...],
            "rows": [
              {"id": 1, "name": "Alice"},
              ...
            ],
            "totalCount": 1000
          }
          ```
    - **Endpoint**: `/api/fs` (File System / Database List)
        - Replaces `listFiles`.
    - **Endpoint**: `/api/tables` (Table List)
        - Replaces `listTables`.


2.  **Update `html_table.go` (Interim)**:
    - We might technically bypass `html_table.go` entirely and write the JSON handler directly in `server.go`.
    - Eventually delete `html_table.go`.

## Phase 2: Leveraging Banquet for Query Parsing
To avoid reimplementing complex URL parsing and SQL generation logic in the React client, we will continue to use the existing `banquet` Go package on the server.

1.  **URL Handling**:
    - The React client will update the browser URL to reflect the current state (dataset, table, filters, sorts) using the Banquet URL format.
    - When the client needs data (e.g., for the grid), it sends the current path/URL to the server's API (e.g., `/api/rows?path=/...`).
    - Alternatively, for the `/rows` endpoint, we can pass the standard grid parameters (`start`, `end`, `sortModel`) and let the server merge these with the `banquet.ParseNested` result derived from the URL context.

2.  **Server-Side Logic**:
    - The server uses `banquet.ParseNested(url)` to get the `Banquet` struct.
    - It generates the SQL query using `common.ConstructSQL(bq)`.
    - **Crucial Step**: The server executes this SQL query.
    - The server returns the **result rows** AND the **generated SQL query** string in the JSON response.
        - Returning the SQL query string allows the client to display it (as debug info or for user verification) without needing to generate it locally.


## Phase 3: React Client Generalization
The current `App.jsx` has hardcoded columns and endpoints. It needs to be dynamic.

1.  **Dynamic Routing**:
    - Install `react-router-dom`: `npm install react-router-dom`.
    - Configure routes:
        - `/` -> File Browser (List of DBs).
        - `/:db` -> Table Browser (List of Tables).
        - `/:db/:table` -> Grid View.

2.  **Dynamic Grid**:
    - Update `App.jsx` (or split into components):
        - Extract `db` and `table` from URL.
        - Fetch column definitions from the server (either via a metadata call or inferred from the first row of response).
        - Pass dynamic `columnDefs` to `<AgGridReact />`.
    - Implement Server-Side Row Model (Infinite Scroll) connecting to the API variables.

## Phase 4: Server Integration & Serving
Embed the React application into the Go binary.

1.  **Build React App**:
    - Run `npm run build` in `react-client/`.
    - Ensure `dist/` is generated.

2.  **Embed Assets**:
    - Use `go:embed` to embed the `react-client/dist` directory.
    - Update `server/server.go` to:
        - Serve static files (js, css) from `dist/assets`.
        - Serve `index.html` for any unknown route (SPA fallback).

## Phase 5: Execution Strategy

1.  **Modify Go Server**: Add the JSON API handler for `/rows`. ensure it respects the requested range and sorting.
2.  **Test API**: Verify the API returns correct JSON for the demo table.
3.  **Modify React Client**: Update `App.jsx` to fetch from the new API and handle dynamic columns.
4.  **Build & Embed**: Configure the file serving in Go.
5.  **Cleanup**: Remove the old HTML templating code.

## Immediate Next Steps (Task List)
- [ ] Modify `server/server.go` to handle `/rows` request and return JSON data.
- [ ] Modify `react-client/src/App.jsx` to accept dynamic columns and use the generic API.
- [ ] Build the client and wire up the static file serving.
