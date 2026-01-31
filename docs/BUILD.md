# Sqliter Build & Development Guide

This project uses [Mage](https://magefile.org/) for build automation. Mage is a Make/Rake-like build tool using Go.

## Prerequisites

- Go 1.25.0 or higher
- Node.js and npm (for React client)
- [Mage](https://magefile.org/) - Install with: `go install github.com/magefile/mage@latest`

## Quick Start

```bash
# Build everything (default target)
mage

# Install to system
mage install

# Run in development mode
mage dev
```

## Available Commands

### Build Commands

- **`mage build`** (default) - Builds the complete sqliter binary with embedded UI
- **`mage buildUI`** - Builds only the React client
- **`mage syncUI`** - Copies built React client to Go embed directory
- **`mage clean`** - Removes all build artifacts

### Development Commands

- **`mage dev`** - Builds and runs the server with sample data
- **`mage relaunch`** - Kills old instances, rebuilds, and starts server in background
- **`mage fmt`** - Formats all Go code
- **`mage lint`** - Runs linters (requires golangci-lint)
- **`mage test`** - Runs all tests

### Installation Commands

- **`mage install`** - Installs sqliter to your system
  - Tries `/usr/local/bin` first
  - Falls back to `~/bin` if no write access
  - Makes binary executable
  - Warns if install directory is not in PATH

- **`mage uninstall`** - Removes sqliter from standard install locations

### Combined Commands

- **`mage all`** - Runs fmt, lint, test, and build in sequence

## Installation Locations

The install command will place the binary in:

1. **macOS/Linux**: `/usr/local/bin/sqliter` (preferred) or `~/bin/sqliter` (fallback)
2. **Windows**: Not yet supported - manually copy `bin/sqliter.exe` to your PATH

If installed to `~/bin`, you may need to add it to your PATH:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/bin:$PATH"
```

## Legacy Scripts

The following bash scripts are still available but deprecated in favor of Mage:

- `react_go_build.sh` - Use `mage build` instead
- `relaunch.sh` - Use `mage relaunch` instead

## Examples

```bash
# Clean build from scratch
mage clean build

# Development workflow
mage dev

# Prepare for release
mage all install

# Uninstall
mage uninstall
```

## Troubleshooting

### "mage: command not found"

Install mage:
```bash
go install github.com/magefile/mage@latest
```

### Permission denied during install

If you don't have write access to `/usr/local/bin`, the installer will automatically use `~/bin`. Make sure `~/bin` is in your PATH.

### React build fails

Ensure you have Node.js and npm installed:
```bash
node --version
npm --version
```

## Project Structure

```
sqliter/
├── magefile.go          # Mage build definitions
├── cmd/sqliter/         # Main application entry point
├── sqliter/             # Core server code
│   └── ui/             # Embedded React UI (generated)
├── react-client/        # React UI source
│   └── dist/           # Built UI (generated)
└── bin/                # Compiled binaries (generated)
```
