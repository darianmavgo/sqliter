# Changelog

## Features Added

### Core Capabilities
- **Embedded Assets**: CSS and JavaScript assets are now embedded directly into the application binary using Go's `embed` package. This eliminates runtime dependencies on external file paths for static assets.
- **Web Navigation**: The file and table browser now automatically generates clickable links:
    - Database files (`.db`, `.sqlite`, etc.) allow navigation into their content.
    - Tables within databases are listed with direct links.
- **Row CRUD Operations**: Added full support for **Create**, **Update**, and **Delete** operations on table rows directly from the web interface.
- **Auto-Redirect**: The server now automatically redirects to the table view if a database contains only a single table, streamlining the user experience.
- **Strict HTML Compliance**: Validated that all generated HTML tags are strictly compliant.

### Project Structure
- **Asset Consolidation**: Centralized all template-related assets (HTML, CSS, JS) into `sqliter/templates/`.
- **Test Suite Enhancements**: Added comprehensive tests for:
    - Browser link navigation flow (`TestBrowserLinksFlow`).
    - CSS file exclusivity (`TestOnlyDefaultCSS`).
    - HTML strictness (`TestHTMLStrictness`).
    - CRUD operations verification (`TestRowCRUD`).

## Features Subtracted

- **Legacy Asset Directories**: Removed the `cssjs/` directory. All assets are now served from the embedded `templates/` directory.
- **Vendor Directory**: Eliminated the `vendor/` directory in favor of Go workspaces/modules.
- **External Stylesheet References**: Removed hardcoded references to external/sample stylesheets (e.g., `/style1/stylesheet.css`) to ensure a consistent, self-contained look and feel.

## Currently Working

- **Web Server**: reliably serves SQLite databases and renders tables with pagination/scrolling support implied by design.
- **Embedded Resource Serving**: The application correctly serves `default.css` and `default.js` from memory.
- **Database Navigation**: Users can traverse from a list of databases to specific tables seamlessly.
- **Data Modification**: Users can add, edit, and remove rows. Changes are persisted to the underlying SQLite database.
- **Automated Verification**: All currently implemented tests (CSS checks, HTML strictness, Link flows, CRUD) are **PASSED**.

## What is Broken / Known Issues

- **Port Flexibility**: The "loop through ports" feature (to find an available port automatically) is not currently present in the main entry point (`cmd/servelocal/main.go`), limiting the server to the configured port only.
- **UI/UX Polish**: While functional, the table editing UI has been noted to have potential jitteriness (e.g., right-click behavior) in previous iterations. No automated tests currently cover UI "feel" or client-side interaction smoothness beyond basic functional correctness.
