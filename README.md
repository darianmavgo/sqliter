# Sqliter - SQLite Database Viewer

A fast, web-based SQLite database viewer with AG Grid for high-performance data exploration.

## Features

- 🚀 **Fast Streaming**: Lazy-loads data using AG Grid's Infinite Row Model
- 📊 **Full-Screen Grid**: Clean, distraction-free interface
- 🔍 **Smart Filtering**: Column filters and sorting
- 📁 **Directory Scanning**: Recursively finds all `.db` and `.sqlite` files
- 🌐 **Banquet URL Support**: Parse and query using Banquet URL syntax
- ⚡ **No CGO**: Pure Go SQLite implementation using `modernc.org/sqlite`

## Quick Start

### Installation

```bash
# Install Mage (build tool)
go install github.com/magefile/mage@latest

# Install sqliter
mage install
```

### Usage

```bash
# View a database
sqliter path/to/database.db

# View a directory (recursively finds all .db files)
sqliter ~/Documents

# No arguments defaults to home directory
sqliter
```

The server will automatically open in your browser at `http://[::1]:PORT`.

## Building from Source

See [docs/BUILD.md](docs/BUILD.md) for detailed build instructions.

Quick build:
```bash
mage build
```

## Development

```bash
# Run in development mode
mage dev

# Clean and rebuild
mage clean build

# Run tests
mage test
```

## Architecture

- **Backend**: Go server with embedded React UI
  - `modernc.org/sqlite` for CGO-free SQLite access
  - `banquet` for URL-based query parsing
  - `mksqlite/converters/filesystem` for directory scanning
  
- **Frontend**: React with AG Grid
  - Infinite Row Model for streaming large datasets
  - Full-screen grid interface
  - Client-side filtering and sorting

## Project Structure

```
sqliter/
├── cmd/sqliter/         # Main application
├── sqliter/             # Server implementation
├── react-client/        # React UI source
├── docs/                # Documentation
└── magefile.go          # Build automation
```

## Documentation

- [Build Guide](docs/BUILD.md) - Build, install, and development commands
- [API Documentation](docs/API.md) - REST API reference (if exists)

## License

See LICENSE file for details.