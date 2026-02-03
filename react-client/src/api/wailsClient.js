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
    }
    
    // Support for OpenDatabase (Specific to Wails usage)
    async openDatabase() {
        return window.go.wails.App.OpenDatabase();
    }

    async getPendingFile() {
        return window.go.wails.App.GetPendingFile();
    }
}
