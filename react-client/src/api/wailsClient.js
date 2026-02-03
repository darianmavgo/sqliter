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
         // Map to Backend ExecuteQuery or structured Query
         // The Go backend for Wails might need a specific struct argument rather than flat params
         // similar to Engine.QueryOptions.
         // Let's assume we pass a JSON string or a struct that matches.
         // Since Wails handles JSON marshaling, we can pass an object.
         
         // We might need to adapt the options to match what the Go side expects.
         // Engine.QueryOptions has { BanquetPath, FilterWhere, SortCol... }
         // The Client query receives { path, start, end, sortCol, ... }
         
         // We'll construct the options object expected by Go
         const queryOpts = {
             BanquetPath: path,
             FilterWhere: "", // Client needs to convert filterModel to SQL if Wails backend doesn't do it?
                              // Or, better, Wails backend uses Same Engine Logic.
                              // Engine.Query expects FilterWhere (SQL).
                              // Server.go handled BuildWhereClause.
                              // So Wails App.go also needs to handle BuildWhereClause if we pass filterModel.
                              // OR we pass filterModel and let Go handle it.
             SortCol: options.sortCol || "",
             SortDir: options.sortDir || "",
             Offset: options.start || 0,
             Limit: (options.end && options.start !== undefined) ? (options.end - options.start) : 0,
             AllowOverride: true,
             SkipTotalCount: !!options.skipTotalCount
         };

         // If filterModel is present, we might need to process it.
         // For now, let's assume we pass it as a separate field 'FilterModelJSON' if we update Go side,
         // or we ignore it if we can't process it here easily without the go helper.
         // Ideally, the Wails App on Go side should expose a method that takes these exact params
         // and calls BuildWhereClause just like Server.go does.
         
         if (options.filterModel) {
            queryOpts.FilterModelJSON = JSON.stringify(options.filterModel);
         }

         const response = await window.go.wails.App.Query(queryOpts);

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
