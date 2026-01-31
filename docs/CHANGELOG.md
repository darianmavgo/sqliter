# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- **Embedded Assets**: React client (HTML, CSS, JS) is now directly embedded into the Go binary using `go:embed`.
- **Web Navigation**:
    - Clickable links for navigating into database files (`.db`, `.sqlite`, etc.).
    - Direct table listing navigation.
- **Row CRUD Operations**: Support for **Create**, **Update**, and **Delete** rows directly from the grid.
- **Auto-Redirect**: Automatically redirects to the table view if a database contains only a single table.
- **Strict HTML Compliance**: Validation for generated HTML tags.
- **Banquet Integration**: Advanced URL-based query parsing for filtering and sorting.
- **Port Flexibility**: Server now automatically finds an available port if the default is taken.

### Changed
- **Asset Consolidation**: All UI assets are built from `react-client` and embedded from `sqliter/ui/`.
- **Project Structure**: Eliminated `vendor/` directory in favor of Go modules.
- **Removed**: Legacy `cssjs/` directory and external stylesheet references.

### Fixed
- **Port Selection**: The entry point now correctly loops/selects a random available port.

## [v0.0.1] - Previous Work
- Initial implementation of SQLite viewer.
- Basic Ag-Grid integration.
