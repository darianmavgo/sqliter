import { DataClient } from './client';

export class WailsClient extends DataClient {
    constructor() {
        super();
        // Ensure Wails runtime is available
        if (!window.runtime) {
            console.warn("Wails runtime not found");
        }
    }

    async listFiles(dir) {
        // Map to Backend ListFiles
        return window.go.wails.App.ListFiles(dir);
    }

    async listTables(db) {
        // Map to Backend ListTables
        return window.go.wails.App.ListTables(db);
    }

    async query(path, options = {}) {
        try {
             const start = options.start !== undefined ? options.start : 0;
             const end = options.end !== undefined ? options.end : 0;

             const limit = (options.end !== undefined) ? (end - start) : 0;

             const queryOpts = {
                 BanquetPath: path,
                 FilterWhere: "",
                 SortCol: options.sortCol || "",
                 SortDir: options.sortDir || "",
                 Offset: start,
                 Limit: limit,
                 AllowOverride: true,
                 SkipTotalCount: !!options.skipTotalCount,
                 ForceZeroLimit: (limit === 0 && options.end !== undefined)
             };

             if (options.filterModel) {
                queryOpts.FilterModelJSON = JSON.stringify(options.filterModel);
             }

             const timerLabel = `[WailsClient] Query ${Date.now()}-${Math.random()}`;
             console.time(timerLabel);
             const response = await window.go.wails.App.Query(queryOpts);
             console.timeEnd(timerLabel);

             // Transform values (array of arrays) back to array of objects for frontend consumption
             if (response.values && response.columns) {
                 const { values, columns } = response;
                 response.rows = values.map(row => {
                     const obj = {};
                     columns.forEach((col, index) => {
                         obj[col] = row[index];
                     });
                     return obj;
                 });
             }

             return response;
        } catch (e) {
            // Report memory usage if available (Chrome only)
            if (window.performance && window.performance.memory) {
                const mem = window.performance.memory;
                console.error("Wails Query Failed. Memory Stats:", {
                    usedJSHeapSize: Math.round(mem.usedJSHeapSize / 1024 / 1024) + ' MB',
                    totalJSHeapSize: Math.round(mem.totalJSHeapSize / 1024 / 1024) + ' MB',
                    jsHeapSizeLimit: Math.round(mem.jsHeapSizeLimit / 1024 / 1024) + ' MB'
                });
            }
            throw e;
        }
    }

    /**
     * streamQuery initiates a streaming query and invokes callbacks as chunks arrive.
     * 
     * @param {string} path - Banquet path
     * @param {object} options - Query options
     * @param {object} callbacks - { onSchema: ({columns, totalCount}) => void, onRows: (rows) => void, onEnd: () => void, onError: (err) => void }
     * @returns {function} cancel function (to be implemented later if backend supports it)
     */
    async streamQuery(path, options = {}, callbacks = {}) {
        const { onSchema, onRows, onEnd, onError } = callbacks;
        
        const start = options.start !== undefined ? options.start : 0;
        const end = options.end !== undefined ? options.end : 0;
        const limit = (options.end !== undefined) ? (end - start) : 0;

        const queryOpts = {
             BanquetPath: path,
             FilterWhere: "",
             SortCol: options.sortCol || "",
             SortDir: options.sortDir || "",
             Offset: start,
             Limit: limit,
             AllowOverride: true,
             SkipTotalCount: !!options.skipTotalCount,
             ForceZeroLimit: (limit === 0 && options.end !== undefined)
         };

         if (options.filterModel) {
            queryOpts.FilterModelJSON = JSON.stringify(options.filterModel);
         }

         try {
             // 1. Generate ID and Subscribe FIRST to avoid race condition
             const queryID = `q-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
             
             // 2. Setup listeners
             let columns = [];
             
             // Handler for Chunks
             const chunkHandler = (chunk) => {
                 // First chunk usually contains columns
                 if (chunk.columns && chunk.columns.length > 0) {
                     columns = chunk.columns;
                     if (onSchema) {
                         onSchema({ 
                             columns: chunk.columns, 
                             totalCount: chunk.totalCount,
                             sql: chunk.sql
                         });
                     }
                 }

                 if (chunk.values && chunk.values.length > 0) {
                     const rows = chunk.values.map(row => {
                         const obj = {};
                         columns.forEach((col, index) => {
                             obj[col] = row[index];
                         });
                         return obj;
                     });
                     if (onRows) onRows(rows);
                 }
                 
                 if (chunk.error && onError) {
                     onError(chunk.error);
                 }
             };

             // Handler for End
             const endHandler = (data) => {
                 console.log(`[WailsClient] Stream finished: ${queryID}`);
                 cleanup();
                 if (onEnd) onEnd();
             };

             const errorHandler = (err) => {
                 console.error(`[WailsClient] Stream error: ${queryID}`, err);
                 cleanup();
                 if (onError) onError(err);
             };

             // 3. Subscribe BEFORE Calling Backend
             window.runtime.EventsOn(`sqliter:stream:chunk:${queryID}`, chunkHandler);
             window.runtime.EventsOn(`sqliter:stream:end:${queryID}`, endHandler);
             window.runtime.EventsOn(`sqliter:stream:error:${queryID}`, errorHandler);

             const cleanup = () => {
                 window.runtime.EventsOff(`sqliter:stream:chunk:${queryID}`);
                 window.runtime.EventsOff(`sqliter:stream:end:${queryID}`);
                 window.runtime.EventsOff(`sqliter:stream:error:${queryID}`);
             };

             // 4. Start the stream (Pass ID to backend)
             console.log(`[WailsClient] Starting stream ID: ${queryID}`);
             window.go.wails.App.StreamQuery(queryOpts, queryID).catch(e => {
                 console.error("Failed to call StreamQuery:", e);
                 onError(e);
                 cleanup();
             });

             return cleanup;

         } catch (e) {
             console.error("Failed to start stream", e);
             if (onError) onError(e);
         }
    }
    
    // Support for OpenDatabase (Specific to Wails usage)
    async openDatabase() {
        return window.go.wails.App.OpenDatabase();
    }

    async getPendingFile() {
        return window.go.wails.App.GetPendingFile();
    }
}
