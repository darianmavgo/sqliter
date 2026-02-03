import { DataClient } from './client';

export class HttpClient extends DataClient {
    constructor(basePath) {
        super();
        this.basePath = basePath || '';
        if (this.basePath.endsWith('/')) {
            this.basePath = this.basePath.slice(0, -1);
        }
    }

    _getUrl(endpoint, params = {}) {
        const url = new URL(window.location.origin + this.basePath + endpoint);
        Object.keys(params).forEach(key => {
            if (params[key] !== undefined && params[key] !== null) {
                url.searchParams.append(key, params[key]);
            }
        });
        // We use relative path for fetch if same origin, but URL object requires base.
        // fetch works fine with full URL.
        return url.toString();
    }

    async listFiles(dir) {
        const res = await fetch(this._getUrl('/sqliter/fs', { dir: dir || '' }));
        const data = await res.json();
        if (data.error) throw new Error(data.error);
        return data;
    }

    async listTables(db) {
        const res = await fetch(this._getUrl('/sqliter/tables', { db }));
        const data = await res.json();
        if (data.error) throw new Error(data.error);
        return data.tables || [];
    }

    async query(path, options = {}) {
        try {
            const params = { path, ...options };
            // handle filterModel specially if passed as object
            if (params.filterModel && typeof params.filterModel === 'object') {
                params.filterModel = JSON.stringify(params.filterModel);
            }

            const res = await fetch(this._getUrl('/sqliter/rows', params));
            const data = await res.json();
            if (data.error) throw new Error(data.error);

            // Transform values (array of arrays) back to array of objects for frontend consumption
            if (data.values && data.columns) {
                const { values, columns } = data;
                data.rows = values.map(row => {
                    const obj = {};
                    columns.forEach((col, index) => {
                        obj[col] = row[index];
                    });
                    return obj;
                });
            }

            return data;
        } catch (e) {
            // Report memory usage if available (Chrome only)
            if (window.performance && window.performance.memory) {
                const mem = window.performance.memory;
                console.error("Query Failed. Memory Stats:", {
                    usedJSHeapSize: Math.round(mem.usedJSHeapSize / 1024 / 1024) + ' MB',
                    totalJSHeapSize: Math.round(mem.totalJSHeapSize / 1024 / 1024) + ' MB',
                    jsHeapSizeLimit: Math.round(mem.jsHeapSizeLimit / 1024 / 1024) + ' MB'
                });
            }
            throw e;
        }
    }
}
