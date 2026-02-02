/**
 * @typedef {Object} FileEntry
 * @property {string} name
 * @property {string} type
 */

/**
 * @typedef {Object} TableInfo
 * @property {string} name
 * @property {string} type
 */

/**
 * @typedef {Object} QueryResult
 * @property {string[]} columns
 * @property {Object[]} rows
 * @property {string} [error]
 * @property {number} [totalCount]
 * @property {string} [sql]
 */

/**
 * Abstract Interface for DataClient
 */
export class DataClient {
    /**
     * @param {string} dir
     * @returns {Promise<FileEntry[]>}
     */
    async listFiles(dir) { throw new Error("Not implemented"); }

    /**
     * @param {string} db
     * @returns {Promise<TableInfo[]>}
     */
    async listTables(db) { throw new Error("Not implemented"); }

    /**
     * @param {string} path
     * @param {Object} options
     * @returns {Promise<QueryResult>}
     */
    async query(path, options) { throw new Error("Not implemented"); }
}
