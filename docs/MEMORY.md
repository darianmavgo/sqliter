# Memory Management and OOM

## Why does Out of Memory (OOM) happen if we have Garbage Collection?

Both Go (server-side) and JavaScript (client-side) utilize Garbage Collection (GC) to manage memory. GC automatically frees up memory that is *no longer referenced* by the application. However, OOM errors can still occur for several reasons:

1.  **Retention (Memory Leaks by Reference):**
    *   If the application logic holds references to objects (e.g., storing millions of rows in a global variable or a long-lived cache), the GC *cannot* free them because they are still "in use" from the runtime's perspective.
    *   In the context of `sqliter`, loading an entire large database table into a slice (`Values [][]interface{}`) creates a massive object that must be held in RAM until the request completes.

2.  **Rate of Allocation:**
    *   If the application allocates memory faster than the GC can identify and free unused memory, the process might hit the system limit before cleanup occurs.

3.  **System Limits:**
    *   Every process has a limit. On 32-bit systems, or inside containers with memory limits (e.g., Docker `memory` limit), the available RAM might be small. Even if code is "correct", trying to load a 1GB dataset into 512MB of RAM will cause an OOM.

## Mitigation in SQLiter

To prevent these issues, we implement:
*   **Pagination/Streaming:** Instead of loading all rows, we load small chunks (e.g., 200 rows).
*   **Buffering Limits:** We limit the number of chunks kept in memory (client-side `maxBlocksInCache`).
*   **Monitoring:** We report memory usage when errors occur to identify "heavy" operations.
