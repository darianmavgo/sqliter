import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import { BrowserRouter, Routes, Route, Link, useParams, useNavigate } from 'react-router-dom';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
import './index.css';

ModuleRegistry.registerModules([AllCommunityModule]);

// No wrapper container needed, using absolute/flex layout on root components


const FileBrowser = ({ path }) => {
  const [rowData, setRowData] = useState([]);

  useEffect(() => {
    document.title = path || 'sqliter';
  }, [path]);
  
  useEffect(() => {
    fetch(`/sqliter/fs?dir=${path || ''}`)
      .then(r => r.json())
      .then(d => {
        if (Array.isArray(d)) {
          setRowData(d);
        } else {
          console.error("API Error or unexpected response:", d);
          setRowData([]);
        }
      })
      .catch(err => {
        console.error("Fetch error:", err);
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
    const navigate = useNavigate();

    useEffect(() => {
        if (db) {
            document.title = db;
        }
    }, [db]);

    useEffect(() => {
        fetch(`/sqliter/tables?db=${db}`)
            .then(r => r.json())
            .then(data => {
                if (data.error) {
                    alert(data.error);
                    return;
                }
                const list = data.tables || [];
                setTables(list);
                if (data.autoRedirectSingleTable && list.length === 1) {
                    navigate(`/${db}/${list[0].name}`, { replace: true });
                }
            });
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

    useEffect(() => {
        if (db && table) {
            const title = `${db}/${table}`;
            document.title = title.length > 80 ? title.substring(title.length - 80) : title;
        }
    }, [db, table]);

    useEffect(() => {
         let path = `/${db}/${table}`;
         if (rest) {
             path += `/${rest}`;
         }
         fetch(`/sqliter/rows?path=${path}&start=0&end=0`)
            .then(r => r.json())
            .then(data => {
                if (data.error) {
                    console.error(data.error);
                    return;
                }
                if (data.columns) {
                    setColDefs(data.columns.map(c => ({ field: c, filter: true, sortable: true, resizable: true })));
                }
            });
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
                let url = `/sqliter/rows?path=${path}&start=${startRow}&end=${endRow}`;
                
                if (sortModel.length > 0) {
                  const { colId, sort } = sortModel[0];
                  url += `&sortCol=${colId}&sortDir=${sort}`;
                }

                fetch(url)
                    .then(resp => resp.json())
                    .then(data => {
                         if (data.error) {
                             wsParams.failCallback();
                             return;
                         }
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

    // Logic to determine what to show
    // We look for a database extension in the path to split DB path from table path.
    // Extensions: .db, .sqlite, .csv.db, .xlsx.db
    const dbMatch = splat.match(/(.*?\.(?:db|sqlite|csv\.db|xlsx\.db))(?:\/|$)(.*)/);

    if (dbMatch) {
        const dbPath = dbMatch[1];
        const restPath = dbMatch[2];

        if (!restPath) {
            // It's just the DB, list tables
            return <TableList db={dbPath} />;
        } else {
             // It's inside a DB
             const parts = restPath.split('/');
             const table = parts[0];
             const rest = parts.slice(1).join('/');
             return <GridView db={dbPath} table={table} rest={rest} />;
        }
    } else {
        // It's a directory
        return <FileBrowser path={splat} />;
    }
}

const App = () => {
  return (
    <BrowserRouter>
        <Routes>
            <Route path="/*" element={<MainRouter />} />
        </Routes>
    </BrowserRouter>
  );
};

export default App;
