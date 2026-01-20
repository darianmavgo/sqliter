# Templates Directory

This directory contains HTML templates and SQL templates used by Mavgo Flight for rendering output and generating database structures.

## HTML Templates

### Table Rendering Templates
Used for displaying query results as HTML tables in web browsers:

#### `head.html`
- **Purpose**: HTML table header template
- **Usage**: Renders table headers with column names
- **Variables**: Receives array of column names
- **Output**: HTML `<thead>` section with sortable column headers

#### `row.html` 
- **Purpose**: HTML table row template
- **Usage**: Renders individual data rows
- **Variables**: Receives array of cell values
- **Output**: HTML `<tr>` section with data cells

#### `foot.html`
- **Purpose**: HTML table footer template  
- **Usage**: Closes table structure and adds JavaScript
- **Output**: Closing `</tbody></table>` tags and page footer

### Page Templates

#### `index.html`
- **Purpose**: Main landing page template
- **Usage**: Default page when browsing root directory
- **Features**: Navigation, file browsing interface

#### `2d.html`
- **Purpose**: 2D visualization template
- **Usage**: Data visualization for numeric datasets
- **Features**: Charts, graphs, interactive displays

#### `sortable.html`
- **Purpose**: Sortable table template
- **Usage**: Enhanced table with client-side sorting
- **Features**: JavaScript-based column sorting

#### `nathancockerill.html`
- **Purpose**: Custom visualization template
- **Usage**: Specialized data display format
- **Features**: Custom styling and layout

## SQL Templates

### Table Creation Templates

#### `createcsvtable.sql`
- **Purpose**: Creates SQLite table structure for CSV files
- **Variables**: 
  - `{{.Headers}}` - Array of column names from CSV header
- **Output**: `CREATE TABLE CSV_Table (...)` statement
- **Usage**: Used by `StageCSVSqlite()` function

#### `createxlsxtable.sql`
- **Purpose**: Creates SQLite table structure for Excel sheets
- **Variables**:
  - `{{.Sheetname}}` - Name of Excel sheet
  - `{{.Headers}}` - Array of column names
- **Output**: `CREATE TABLE {{.Sheetname}} (...)` statement
- **Usage**: Used by `StageXlsxSqlite()` function

### Data Insertion Templates

#### `insertrow.sql`
- **Purpose**: Generates parameterized INSERT statements
- **Variables**:
  - `{{.Table}}` - Target table name
  - `{{.Columns}}` - Array of column names
- **Output**: `INSERT INTO {{.Table}} (...) VALUES (...)`
- **Usage**: Bulk data insertion for CSV and Excel processing

## Template Usage

### HTML Template Processing
```go
// Load and execute template
tmpl, err := template.ParseFiles("templates/head.html")
if err != nil {
    log.Println("template failed", err)
    return
}
tmpl.Execute(w, columnNames)
```

### SQL Template Processing  
```go
// Generate CREATE TABLE statement
tmpl, err := template.ParseFiles("templates/createcsvtable.sql")
tableString := &strings.Builder{}
args := struct {
    Headers []string
}{
    Headers: headers,
}
tmpl.Execute(tableString, args)
```

## Template Variables

### Common Variables
- `{{.Headers}}` - Column names array
- `{{.Table}}` - Table name string
- `{{.Columns}}` - Column definitions
- `{{.Sheetname}}` - Excel sheet name

### HTML-Specific Variables
- Column data arrays for table rendering
- CSS classes for styling
- JavaScript for interactivity

## Output Examples

### HTML Table Output
```html
<table class="sortable">
  <thead>
    <tr>
      <th>Name</th>
      <th>Age</th>
      <th>City</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>John Doe</td>
      <td>30</td>
      <td>New York</td>
    </tr>
  </tbody>
</table>
```

### SQL Output
```sql
CREATE TABLE CSV_Table (
  name TEXT,
  age TEXT,
  city TEXT
);

INSERT INTO CSV_Table (name, age, city) 
VALUES (:name, :age, :city);
```

## Styling and Assets

### CSS Integration
- Templates include Bootstrap CSS for responsive design
- Custom CSS for data table styling
- Print-friendly styles for reports

### JavaScript Integration
- Client-side sorting functionality
- Interactive table features
- Data visualization libraries

## Customization

### Adding New Templates
1. Create template file in `templates/` directory
2. Define template variables and structure
3. Reference template in Go code:
   ```go
   tmpl, err := template.ParseFiles("templates/mytemplate.html")
   ```

### Modifying Existing Templates
- Edit HTML structure and styling
- Add new template variables as needed
- Test with sample data to verify output

## Performance Considerations

- Templates are parsed for each request (could be optimized)
- Large result sets stream through row template efficiently
- SQL templates generate optimized database operations

## Security Notes

- Templates should escape user input to prevent XSS
- SQL templates use parameterized queries to prevent injection
- File paths validated before template access

## Integration

Templates integrate with:
- **File handlers**: Generate output for different file types
- **Database operations**: Create tables and insert data
- **Web interface**: Provide user-friendly data display
- **Export functionality**: Generate different output formats

## File Structure

```
templates/
├── README.md              # This file
├── head.html             # Table header template
├── row.html              # Table row template  
├── foot.html             # Table footer template
├── index.html            # Main page template
├── 2d.html               # 2D visualization template
├── sortable.html         # Sortable table template
├── nathancockerill.html  # Custom visualization
├── createcsvtable.sql    # CSV table creation
├── createxlsxtable.sql   # Excel table creation
└── insertrow.sql         # Row insertion template
```

The templates directory provides the presentation layer for Mavgo Flight, enabling rich HTML output and efficient database operations through reusable template components. 