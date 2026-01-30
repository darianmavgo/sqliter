import React, { useState, useEffect, useCallback } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import { BrowserRouter, Routes, Route, Link, useParams, useNavigate } from 'react-router-dom';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
import './index.css';

ModuleRegistry.registerModules([AllCommunityModule]);

// No wrapper container needed, using absolute/flex layout on root components


const FileList = () => {
  const [rowData, setRowData] = useState([]);
  
  useEffect(() => {
    fetch('/sqliter/fs')
      .then(r => r.json())
      .then(d => setRowData(d || []));
  }, []);

  const [colDefs] = useState([
    { 
        field: "name", 
        headerName: "Database Name", 
        flex: 1,
        cellRenderer: (params) => {
            return params.value ? <Link to={`/${params.value}`} style={{color: '#61dafb'}}>{params.value}</Link> : null;
        }
    },
    { field: "type", width: 150 }
  ]);

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

const TableList = () => {
    const { db } = useParams();
    const [tables, setTables] = useState([]);
    const navigate = useNavigate();

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

    const [colDefs] = useState([
        { 
            field: "name", 
            headerName: "Table Name", 
            flex: 1,
            cellRenderer: (params) => {
                return params.value ? <Link to={`/${db}/${params.value}`} style={{color: '#61dafb'}}>{params.value}</Link> : null;
            }
        },
        { field: "type", width: 150 }
    ]);

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

const GridView = () => {
    const params = useParams();
    const db = params.db;
    const table = params.table;
    const rest = params["*"];

    const [colDefs, setColDefs] = useState([]);

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


const App = () => {
  return (
    <BrowserRouter>
        <Routes>
            <Route path="/" element={<FileList />} />
            <Route path="/:db" element={<TableList />} />
            <Route path="/:db/:table/*" element={<GridView />} />
        </Routes>
    </BrowserRouter>
  );
};

export default App;
