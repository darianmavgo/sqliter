export namespace sqliter {
	
	export class FileEntry {
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class QueryOptions {
	    BanquetPath: string;
	    FilterWhere: string;
	    SortCol: string;
	    SortDir: string;
	    Offset: number;
	    Limit: number;
	    AllowOverride: boolean;
	    SkipTotalCount: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QueryOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BanquetPath = source["BanquetPath"];
	        this.FilterWhere = source["FilterWhere"];
	        this.SortCol = source["SortCol"];
	        this.SortDir = source["SortDir"];
	        this.Offset = source["Offset"];
	        this.Limit = source["Limit"];
	        this.AllowOverride = source["AllowOverride"];
	        this.SkipTotalCount = source["SkipTotalCount"];
	    }
	}
	export class QueryResult {
	    columns: string[];
	    rows: any[];
	    totalCount: number;
	    sql: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.totalCount = source["totalCount"];
	        this.sql = source["sql"];
	    }
	}
	export class TableInfo {
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new TableInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}

}

