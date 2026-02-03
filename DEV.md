# Development Documentation

## Logging

### Server-Side (Go Backend)
The Go backend logs to **standard output** (`stdout`).

*   **During Development (`wails dev`)**:
    *   Logs appear directly in the terminal where you started the application.
    *   Look for output from `fmt.Println` (e.g., `Received OpenFile: ...`) or `log.Println`.

*   **In Production (Built App)**:
    *   Logs are **not** saved to a file by default.
    *   To view logs for the built application, you must launch the executable from a terminal:
        ```bash
        ./build/bin/sqliter.app/Contents/MacOS/sqliter
        ```

### Client-Side (React Frontend)
The frontend uses standard browser logging (`console.log`, `console.error`, etc.).

*   **Inside the Desktop App**:
    *   Right-click anywhere in the app window (or use the configured keybinding, often F12 or Cmd+Option+I on Mac, though right-click is most reliable in dev builds).
    *   Select **Inspect Element**.
    *   Navigate to the **Console** tab.

*   **In a Web Browser**:
    *   If you are running the app with `wails dev` and access the **DevServer URL** (e.g., `http://localhost:34115`), use your browser's standard Developer Tools (F12 or Cmd+Option+I).

---

## Testing

### Client-Side Tests
Currently, there are **no automated client-side tests** (e.g., Jest, Vitest) configured for the `react-client`.

*   The `react-client/package.json` contains a `d` script for linting (`npm run lint`), but no `test` script.
*   Testing is primarily done manually by interacting with the UI.

### Server-Side Tests
Automated tests for the Go backend (including Wails app logic) are located in the `tests/` directory.

**To run server-side tests:**

1.  Navigate to the project root.
2.  Run the standard Go test command:
    ```bash
    go test ./tests/...
    ```
