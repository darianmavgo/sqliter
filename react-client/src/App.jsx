import React, { useState, useEffect, useCallback } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import { BrowserRouter, Routes, Route, Link, useParams, useNavigate } from 'react-router-dom';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
import './index.css';

ModuleRegistry.registerModules([AllCommunityModule]);

const Container = ({ children }) => (
  <div style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column', backgroundColor: '#1e1e1e', color: '#e0e0e0' }}>
    {children}
  </div>
);

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
      <div className="ag-theme-alpine-dark" style={{ width: '100%', height: '100%' }}>
          <AgGridReact
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
                setTables(data || []);
            });
    }, [db]);

    return (
        <div style={{ padding: '20px' }}>
          <h1>Tables in {db}</h1>
          <Link to="/" style={{color: '#888', marginBottom: '10px', display: 'block'}}>← Back</Link>
          <ul>
            {tables.map(t => (
               <li key={t.name}>
                 <Link to={`/${db}/${t.name}`} style={{color: '#61dafb'}}>{t.name}</Link> <span style={{color: '#666'}}>({t.type})</span>
               </li>
            ))}
          </ul>
        </div>
    )
}

const GridView = () => {
    const { db, table } = useParams();
    const [colDefs, setColDefs] = useState([]);
    const [sqlDebug, setSqlDebug] = useState("");

    // Fetch simple metadata/columns first
    useEffect(() => {
         // We fetch a 0-row result to get columns
         const path = `/${db}/${table}`;
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
                if (data.sql) {
                    setSqlDebug(data.sql);
                }
            });
    }, [db, table]);

    const onGridReady = useCallback((params) => {
        const dataSource = {
            rowCount: undefined,
            getRows: (wsParams) => {
                const { startRow, endRow, sortModel } = wsParams;
                const path = `/${db}/${table}`;
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
                         setSqlDebug(data.sql); // Update debug SQL
                         wsParams.successCallback(data.rows, data.totalCount);
                    })
                    .catch(err => {
                        console.error(err);
                        wsParams.failCallback();
                    })
            }
        };
        params.api.setGridOption('datasource', dataSource);
    }, [db, table]);

    return (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <div style={{ padding: '10px', borderBottom: '1px solid #333', display: 'flex', gap: '10px', alignItems: 'center' }}>
                <Link to={`/${db}`} style={{color: '#888'}}>← Back</Link>
                <span>{db} / <b>{table}</b></span>
                <span style={{marginLeft: 'auto', fontSize: '10px', color: '#555', fontFamily: 'monospace'}}>{sqlDebug}</span>
            </div>
            <div className="ag-theme-alpine-dark" style={{ flex: 1, width: '100%' }}>
                <AgGridReact
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
        </div>
    );
};


const App = () => {
  return (
    <BrowserRouter>
        <Container>
            <Routes>
                <Route path="/" element={<FileList />} />
                <Route path="/:db" element={<TableList />} />
                <Route path="/:db/:table" element={<GridView />} />
            </Routes>
        </Container>
    </BrowserRouter>
  );
};

export default App;
