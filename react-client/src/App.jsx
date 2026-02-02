import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import { BrowserRouter, Routes, Route, Link, useParams, useNavigate } from 'react-router-dom';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
import './index.css';
import { HttpClient } from './api/httpClient';
import { WailsClient } from './api/wailsClient';

ModuleRegistry.registerModules([AllCommunityModule]);

// Initialize Client
const initializeClient = () => {
    // Check for Wails environment
    if (window.go && window.runtime) {
        console.log("Using Wails Client");
        return new WailsClient();
    }
    // Default to HTTP
    const config = window.SQLITER_CONFIG || {};
    console.log("Using Http Client", config);
    return new HttpClient(config.basePath);
};

const client = initializeClient();

const FileBrowser = ({ path }) => {
  const [rowData, setRowData] = useState([]);
  const [error, setError] = useState(null);
  
  useEffect(() => {
    setError(null);
    client.listFiles(path || '')
      .then(d => {
        if (Array.isArray(d)) {
          setRowData(d);
        } else {
          console.error("API Error or unexpected response:", d);
          setError("Unexpected response from server");
          setRowData([]);
        }
      })
      .catch(err => {
        console.error("Fetch error:", err);
        setError(err.message || "Network error");
        setRowData([]);
      });
  }, [path]);

  const colDefs = useMemo(() => [
    { 
        field: "name", 
        headerName: "Name",
        flex: 1,
        cellRenderer: (params) => {
            const val = params.value;
            if (!val) return null;
            const fullPath = path ? `${path}/${val}` : val;
            return <Link to={`/${fullPath}`} style={{color: '#61dafb'}}>{val}</Link>;
        }
    },
    { field: "type", width: 150 }
  ], [path]);

  if (error) {
      return (
          <div style={{ padding: '20px', color: '#ff6b6b', background: '#2c3e50', height: '100%' }}>
              <h3>Error loading files</h3>
              <p>{error}</p>
          </div>
      );
  }

  return (
      <div style={{ width: '100%', height: '100%' }} className="ag-theme-alpine-dark">
        <AgGridReact
            className="ag-theme-alpine-dark"
            theme="legacy"
            rowData={rowData}
            columnDefs={colDefs}
            defaultColDef={{sortable: true, filter: true, resizable: true}}
            rowHeight={32}
            headerHeight={32}
        />
      </div>
  );
};

const TableList = ({ db }) => {
    const [tables, setTables] = useState([]);
    const [activeTable, setActiveTable] = useState(null);
    const navigate = useNavigate();

    useEffect(() => {
        setActiveTable(null);
        client.listTables(db)
            .then(list => {
                setTables(list);
                // Auto-redirect logic could be handled here if we passed config to client or checked list length
                // For now, keeping it simple.
                if (list.length === 1) {
                     setActiveTable(list[0].name);
                }
            })
            .catch(err => alert(err.message));
    }, [db, navigate]);

    const colDefs = useMemo(() => [
        { 
            field: "name", 
            headerName: "Table Name", 
            flex: 1,
            cellRenderer: (params) => {
                return params.value ? <Link to={`/${db}/${params.value}`} style={{color: '#61dafb'}}>{params.value}</Link> : null;
            }
        },
        { field: "type", width: 150 }
    ], [db]);

    if (activeTable) {
        return <GridView db={db} table={activeTable} rest="" />;
    }

    return (
        <div style={{ width: '100%', height: '100%' }} className="ag-theme-alpine-dark">
            <AgGridReact
                className="ag-theme-alpine-dark"
                theme="legacy"
                rowData={tables}
                columnDefs={colDefs}
                defaultColDef={{sortable: true, filter: true, resizable: true}}
                rowHeight={32}
                headerHeight={32}
            />
        </div>
    );
}

const GridView = ({ db, table, rest }) => {
    const [colDefs, setColDefs] = useState([]);

    // Fetch initial column defs
    useEffect(() => {
         let path = `/${db}/${table}`;
         if (rest) {
             path += `/${rest}`;
         }
         client.query(path, { start: 0, end: 0 })
            .then(data => {
                if (data.columns) {
                    setColDefs(data.columns.map(c => ({ field: c, filter: true, sortable: true, resizable: true })));
                }
            })
            .catch(console.error);
    }, [db, table, rest]);

    const onGridReady = useCallback((params) => {
        const dataSource = {
            rowCount: undefined,
            getRows: (wsParams) => {
                const { startRow, endRow, sortModel } = wsParams;
                let path = `/${db}/${table}`;
                if (rest) {
                    path += `/${rest}`;
                }
                
                const apiParams = {
                    start: startRow,
                    end: endRow,
                    skipTotalCount: true
                };
                
                if (sortModel.length > 0) {
                  const { colId, sort } = sortModel[0];
                  apiParams.sortCol = colId;
                  apiParams.sortDir = sort;
                }

                if (wsParams.filterModel && Object.keys(wsParams.filterModel).length > 0) {
                    apiParams.filterModel = wsParams.filterModel;
                }

                client.query(path, apiParams)
                    .then(data => {
                         wsParams.successCallback(data.rows, data.totalCount);
                    })
                    .catch(err => {
                        console.error(err);
                        wsParams.failCallback();
                    })
            }
        };
        params.api.setGridOption('datasource', dataSource);
    }, [db, table, rest]);

    return (
        <div style={{ width: '100%', height: '100%' }} className="ag-theme-alpine-dark">
            <AgGridReact
                className="ag-theme-alpine-dark"
                theme="legacy"
                columnDefs={colDefs}
                rowModelType={'infinite'}
                onGridReady={onGridReady}
                cacheBlockSize={100}
                maxBlocksInCache={10}
                rowHeight={32}
                headerHeight={32}
            />
        </div>
    );
};

const MainRouter = () => {
    const params = useParams();
    const splat = params["*"] || "";

    useEffect(() => {
        const title = splat || 'sqliter';
        document.title = title.length > 80 ? title.substring(title.length - 80) : title;
    }, [splat]);

    const dbMatch = splat.match(/(.*?\.(?:db|sqlite|csv\.db|xlsx\.db))(?:\/|$)(.*)/);

    if (dbMatch) {
        const dbPath = dbMatch[1];
        const restPath = dbMatch[2];

        if (!restPath) {
            return <TableList db={dbPath} />;
        } else {
             const parts = restPath.split('/');
             const table = parts[0];
             const rest = parts.slice(1).join('/');
             return <GridView db={dbPath} table={table} rest={rest} />;
        }
    } else {
        return <FileBrowser path={splat} />;
    }
}

const App = () => {
  const config = window.SQLITER_CONFIG || {};
  const basePath = config.basePath || ''; 
  
  return (
    <BrowserRouter basename={basePath}>
        <Routes>
            <Route path="/*" element={<MainRouter />} />
        </Routes>
    </BrowserRouter>
  );
};

export default App;
