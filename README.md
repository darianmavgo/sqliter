# Ag-Grid Client with SQLite Streaming

This project implements an Ag-Grid client that streams data from a large SQLite database without locking the UI.

## Architecture

- **Server (`go-server`)**: A lightweight Go server using `modernc.org/sqlite` (no CGO) to serve rows from `Index.sqlite` via a simple REST API. It handles pagination and sorting.
- **Client (`react-client`)**: A React application using `ag-grid-react` and `ag-grid-community` (v32+). It uses the Infinite Row Model to fetch data in chunks as the user scrolls, ensuring high performance even with large datasets.

## Features

1. **Sticky Headers**: Native Ag-Grid support properly configured.
2. **Tight Column Sizing**: optimized `rowHeight` and `headerHeight` and tight column definitions.
3. **Column Sort**: Server-side sorting implemented for all columns.
4. **Streaming Data**: Uses Infinite Row Model to lazy-load data.

## How to Run

### 1. Start the Data Server
```bash
cd go-server
go run main.go
# Server listens on http://localhost:8080/rows
```

### 2. Start the Client
```bash
cd react-client
npm install
npm run dev
# Client listens on http://localhost:5173
```

## Troubleshooting
- If you see "No AG Grid modules are registered", ensure `ModuleRegistry.registerModules([AllCommunityModule])` is called in `App.jsx`.
- If data doesn't load, check that the Go server is running and `Index.sqlite` is accessible.