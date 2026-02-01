# SQLiter

**SQLiter** is a lightweight, high-performance SQLite database viewer and editor. It combines a robust Go server with a modern React frontend (using Ag-Grid) to provide a seamless data browsing experience.

## Features Supported

*   **High-Peformance Streaming**: integration with Ag-Grid's Infinite Row Model allows browsing million-row datasets with zero lag.
*   **Single-Binary Distribution**: The entire React frontend is embedded in the Go binary. No separate node server required for deployment.
*   **CRUD Operations**: Full support for Creating, Reading, Updating, and Deleting rows via the UI.
*   **Deep Linking**: Every query, filter, and view state is encoded in the URL, making data shareable.
*   **Advanced Querying**: Leverages **Banquet** to support complex filtering, sorting, and column selection via URL parameters.
*   **Flexible Inputs**: Supports local files, directories, and remote URLs (streaming SQLite over HTTP).

## Area of Responsibility

SQLiter is the **Presentation Layer**. It is responsible for:
1.  **Visualization**: Rendering raw data into interactive, human-readable tables.
2.  **Interaction**: providing the user interface for exploring, filtering, and editing data.
3.  **Delivery**: Serving the client-side application to the browser.
It acts as the "Face" of the data ecosystem.

## Scope (What it explicitly doesn't do)

*   **No Data Ingestion/Conversion**: SQLiter expects data to already be in valid SQLite format (or a compatible directory structure). It does not convert CSVs or scrape websites (that is the job of `mksqlite`).
*   **Not a Generic Web Server**: While it includes an HTTP server, it is specialized for serving database content and API endpoints. It is not designed to host general-purpose websites.
*   **No User Management**: It assumes it is running in a trusted environment or behind a gateway (like `flight3` or PocketBase) that handles auth; it does not implement its own user/role system.

## Usage

```bash
# View a database
sqliter my_data.db

# View a remote file
sqliter https://example.com/data.sqlite
```
