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
    // Optimization: Don't list root ("") or "/" by default to avoid scanning slow volumes
    if (!path || path === "/" || path.trim() === "") {
        setRowData([]);
        return;
    }

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

  if (!path || path === "/" || path.trim() === "") {
      return (
        <div style={{ padding: '40px', textAlign: 'center', color: '#888' }}>
            <h2>Welcome to SQLiter</h2>
            <p>Open a database file to get started</p>
            {window.go && (
                <button 
                    onClick={() => client.openDatabase().then(p => p && (window.location.hash = `/${p}`))}
                    style={{ background: '#61dafb', border: 'none', borderRadius: '4px', padding: '10px 20px', cursor: 'pointer', fontWeight: 'bold', fontSize: '16px', marginTop: '20px', color: '#282c30' }}
                >
                    Open Database
                </button>
            )}
        </div>
      );
  }

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
    const initialData = React.useRef(null);

    // Fetch initial column defs and first 50 rows
    useEffect(() => {
         let path = `/${db}/${table}`;
         if (rest) {
             path += `/${rest}`;
         }
         // Request first 50 rows immediately to get columns + data
         client.query(path, { start: 0, end: 50, skipTotalCount: true })
            .then(data => {
                if (data.columns) {
                    setColDefs(data.columns.map(c => ({ field: c, filter: true, sortable: true, resizable: true })));
                }
                // Cache the initial rows to feed the grid immediately
                initialData.current = data.rows;
            })
            .catch(console.error);
    }, [db, table, rest]);

    const onGridReady = useCallback((params) => {
        const dataSource = {
            rowCount: undefined,
            getRows: (wsParams) => {
                const { startRow, endRow, sortModel } = wsParams;
                
                // Optimization: Use pre-fetched data for the first block if available
                if (startRow === 0 && initialData.current && (!sortModel.length) && (!wsParams.filterModel || Object.keys(wsParams.filterModel).length === 0)) {
                    console.log("Using initial data for first block");
                    const rows = initialData.current;
                    initialData.current = null; // Clear it so we don't reuse it inappropriately
                    // If rows < 50, we know the exact count
                    const lastRow = rows.length < 50 ? rows.length : -1;
                    wsParams.successCallback(rows, lastRow);
                    return;
                }

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
                cacheBlockSize={50}
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

const Navbar = () => {
    const navigate = useNavigate();
    const params = useParams();
    const splat = params["*"] || "";
    const [inputValue, setInputValue] = useState(splat);

    useEffect(() => {
        setInputValue(splat);
    }, [splat]);

    const handleKeyDown = (e) => {
        if (e.key === 'Enter') {
            const path = inputValue.startsWith('/') ? inputValue : `/${inputValue}`;
            navigate(path);
        }
    };

    // Check if current path is a table
    const dbMatch = splat.match(/(.*?\.(?:db|sqlite|csv\.db|xlsx\.db))(?:\/|$)(.*)/);
    const isTable = dbMatch && dbMatch[2];

    const runSpeedTest = () => {
        if (!isTable) return;
        const path = `/${splat}`;
        console.log(`[SpeedTest] Starting test for ${path}...`);
        const start = performance.now();

        // Match GridView's initial load: start:0, end:50, skipTotalCount: true
        client.query(path, { start: 0, end: 50, skipTotalCount: true })
            .then(data => {
                const end = performance.now();
                const duration = end - start;
                const msg = `[SpeedTest] Fetched ${data.rows ? data.rows.length : 0} rows in ${duration.toFixed(2)}ms`;
                console.log(msg);
                alert(msg);
            })
            .catch(err => {
                const end = performance.now();
                console.error(`[SpeedTest] Failed after ${(end - start).toFixed(2)}ms`, err);
                alert(`[SpeedTest] Failed: ${err.message}`);
            });
    };

    return (
        <div style={{ 
            height: '40px', 
            background: '#20232a', 
            display: 'flex', 
            alignItems: 'center', 
            padding: '0 10px', 
            borderBottom: '1px solid #333',
            gap: '10px'
        }}>
            <button 
                onClick={() => navigate(-1)} 
                style={{ background: 'none', border: 'none', color: '#61dafb', cursor: 'pointer', fontSize: '18px' }}
                title="Back"
            >
                ←
            </button>
            <button 
                onClick={() => navigate('/')} 
                style={{ background: 'none', border: 'none', color: '#61dafb', cursor: 'pointer', fontWeight: 'bold' }}
            >
                SQLiter
            </button>
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', background: '#282c34', borderRadius: '4px', padding: '0 8px' }}>
                <span style={{ color: '#888', marginRight: '5px' }}>URI</span>
                <input 
                    type="text" 
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    style={{ 
                        flex: 1, 
                        background: 'none', 
                        border: 'none', 
                        color: '#fff', 
                        outline: 'none', 
                        padding: '5px 0',
                        fontSize: '13px',
                        fontFamily: 'monospace'
                    }}
                />
            </div>
            {isTable && (
                 <button
                    onClick={runSpeedTest}
                    style={{ background: '#e74c3c', border: 'none', borderRadius: '4px', padding: '4px 10px', cursor: 'pointer', fontWeight: 'bold', color: '#fff', fontSize: '12px' }}
                    title="Run Speed Test (50 rows)"
                >
                    Test Speed
                </button>
            )}
            {window.go && (
                <button 
                    onClick={() => client.openDatabase().then(p => p && navigate(`/${p}`))}
                    style={{ background: '#61dafb', border: 'none', borderRadius: '4px', padding: '4px 10px', cursor: 'pointer', fontWeight: 'bold' }}
                >
                    Open File
                </button>
            )}
        </div>
    );
};

const InnerApp = () => {
  const navigate = useNavigate();

  useEffect(() => {
    if (window.runtime && window.runtime.EventsOn) {
        console.log("Setting up Wails event listeners");
        
        const handleFile = (filePath) => {
            console.log("Processing file path:", filePath);
            if (!filePath) return;
            const target = filePath.startsWith('/') ? filePath : `/${filePath}`;
            navigate(target);
        };

        // Handle files opened via macOS Finder while app is running
        window.runtime.EventsOn("open-file", handleFile);

        // Check for file opened at startup
        client.getPendingFile?.().then(handleFile);

        return () => window.runtime.EventsOff("open-file");
    }
  }, [navigate]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: '#282c34', color: '#fff' }}>
        <Routes>
            <Route path="/*" element={<><Navbar /><div style={{ flex: 1, overflow: 'hidden' }}><MainRouter /></div></>} />
        </Routes>
    </div>
  );
}

const App = () => {
  const config = window.SQLITER_CONFIG || {};
  const basePath = config.basePath || ''; 
  
  return (
    <BrowserRouter basename={basePath}>
        <InnerApp />
    </BrowserRouter>
  );
};

export default App;
