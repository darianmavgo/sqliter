# SQLiter

**SQLiter** is a lightweight, high-performance SQLite database viewer and editor. It combines a robust Go server with a modern React frontend (using Ag-Grid) to provide a seamless data browsing experience.

## ✨ Features

- **🚀 Streaming Performance**: Uses Ag-Grid's Infinite Row Model to handle large datasets efficiently without loading everything into memory.
- **📦 Single Binary**: The React frontend is embedded into the Go binary, making deployment as simple as copying a single executable.
- **🛠️ CRUD Operations**: Create, update, and delete rows directly from the interface.
- **🔗 Deep Linking**: Every view (database, table, query) is URL-addressable, making it easy to share specific data views.
- **📂 Flexible Input**: Open local files, browse directories, or stream databases directly from a URL.
- **⚡ Zero Configuration**: Just run the binary and point it to your data.
- **🔍 Advanced Querying**: Supports complex filtering and sorting via URL parameters (powered by [Banquet](https://github.com/darianmavgo/banquet)).

## 📦 Installation

### Using Go Install
If you have Go installed, you can install the latest version directly:

```bash
go install github.com/darianmavgo/sqliter/cmd/sqliter@latest
```

### Building from Source

1.  **Prerequisites**:
    *   Go 1.24+
    *   Node.js & npm (for building the frontend)

2.  **Clone and Build**:
    ```bash
    git clone https://github.com/darianmavgo/sqliter.git
    cd sqliter
    ./react_go_build.sh
    ```
    This script builds the React client, embeds it into the Go server, and produces a binary in `bin/sqliter`.

## 🚀 Usage

Run `sqliter` with a file, directory, or URL:

### Open a Local Database
```bash
sqliter my_database.db
```

### Browse a Directory
```bash
sqliter ~/Documents/databases
```

### Open a Remote Database
```bash
sqliter https://example.com/datasets/large_data.sqlite
```

The application will automatically find an available port, start the server, and open your default browser.

## 🛠️ Development

The project consists of two main parts:
1.  **Backend (`/sqliter`, `/cmd`)**: A Go HTTP server using `modernc.org/sqlite` (CGO-free).
2.  **Frontend (`/react-client`)**: A React application using Vite and Ag-Grid.

### Running in Development Mode

To work on the frontend with hot-reload:

1.  **Start the Go Server** (serving data):
    ```bash
    go run cmd/sqliter/main.go sample_data/
    ```
    *Note the port it starts on (e.g., 8080).*

2.  **Start the React Client**:
    ```bash
    cd react-client
    npm install
    npm run dev
    ```
    *Open the Vite URL (e.g., http://localhost:5173).*

    *Note: You may need to configure the React app to proxy requests to the Go server port if not automatically handled.*

### Building the Release

Use the provided script to build the full distribution:
```bash
./react_go_build.sh
```

## 🏗️ Architecture

-   **Server**: Go (Standard Library + `modernc.org/sqlite` + `banquet` for query parsing).
-   **Client**: React + Ag-Grid Community + Vite.
-   **Communication**: REST API (`/sqliter/...`) serving JSON.
-   **Embedding**: `go:embed` is used to bundle the compiled React assets (`dist/`) into the Go binary.

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.
