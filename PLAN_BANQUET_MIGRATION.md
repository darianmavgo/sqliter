# Plan: Port Banquet Logic to Sqliter

## Objective
Implement `banquet` URL parsing and query execution logic in `sqliter`'s Go server, enabling sophisticated querying via URL parameters (filtering, sorting, column selection) similar to `flight3`, but strictly for existing local SQLite files and returning JSON data instead of HTML.

## Constraints
1.  **No HTML/CSS/JS Generation**: The server must return JSON.
2.  **No PocketBase**: Remove all dependencies on PocketBase `core.RequestEvent`.
3.  **No Mksqlite**: Read-only access to existing `.sqlite` or `.db` files. No conversion of CSV/Excel/etc.
4.  **Local Filesystem Only**: Serve files from `~/Documents`.

## Architecture

### 1. Dependencies
-   Add `github.com/darianmavgo/banquet` to `go.mod`.

### 2. URL Structure
The server will handle requests with the Banquet format:
```
GET /<DatasetPath>/<OptionalTable>?select=...&where=...&limit=...
```
Example:
`GET /MyData.sqlite/users?select=id,name&where=age>21`

### 3. Backend Implementation (`main.go`)

We will introduce a new handler `handleBanquet` that captures wildcard paths (or specific `/api` paths if preferred, but Banquet usually takes the root).

**Logic Flow:**
1.  **URL Parsing**:
    -   Extract path from request.
    -   Use `banquet.ParseNested(path)` to get a `Banquet` struct containing:
        -   `DataSetPath` (e.g., `folder/file.sqlite`)
        -   `Table` (e.g., `users`)
        -   Query modifiers (`Select`, `Where`, `OrderBy`, `Limit`, `Offset`, etc.)

2.  **Path Resolution**:
    -   Root directory: `~/Documents`.
    -   Full Path = `~/Documents` + `b.DataSetPath`.
    -   Verify file existence. If missing -> 404.

3.  **Table Inference** (if `b.Table` is empty):
    -   Connect to SQLite DB.
    -   Query `sqlite_master`.
    -   If table `tb0` exists -> use `tb0`.
    -   If only 1 table exists -> use it.
    -   Else -> default to `sqlite_master` (show list of tables).

4.  **Query Generation**:
    -   Reimplement `buildSQLQuery` from `flight3` (adapting as needed for `sql`).
    -   Construct `SELECT ... FROM ... WHERE ...` string.

5.  **Execution**:
    -   Execute the query.
    -   Scan rows into a generic map (`[]map[string]interface{}`).

6.  **Response**:
    -   Return JSON:
        ```json
        {
          "rows": [...],
          "columns": ["col1", "col2"],
          "totalCount": 100, // Optional: might require separate COUNT(*) query
          "sql": "SELECT ...",
          "banquet": { ... } // Debug info
        }
        ```

### 4. Integration
-   We will likely keep specific endpoints (`/sqliter/logs`, `/sqliter/fs`) for system functions.
-   The "browsing" endpoints (`/sqliter/rows`) can be replaced or augmented by this logic.

## Steps
1.  Modify `main.go`.
2.  Implement `handleBanquet` and `buildSQLQuery`.
3.  Register handler.
4.  Test with a curl request.
