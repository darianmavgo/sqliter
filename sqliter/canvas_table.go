//go:build js && wasm

package sqliter

import (
	"database/sql"
	"fmt"
	"syscall/js"
)

// CanvasTableRenderer renders SQL query results to an HTML5 canvas using WASM
type CanvasTableRenderer struct {
	canvas       js.Value
	ctx          js.Value
	rows         [][]string
	columns      []string
	scrollY      int
	scrollX      int
	rowHeight    int
	headerHeight int
	colWidths    []int
	canvasWidth  int
	canvasHeight int
}

// NewCanvasTableRenderer creates a new canvas-based table renderer
func NewCanvasTableRenderer(canvasID string) (*CanvasTableRenderer, error) {
	document := js.Global().Get("document")
	canvas := document.Call("getElementById", canvasID)

	if canvas.IsNull() {
		return nil, fmt.Errorf("canvas element not found: %s", canvasID)
	}

	ctx := canvas.Call("getContext", "2d")
	if ctx.IsNull() {
		return nil, fmt.Errorf("failed to get 2d context")
	}

	return &CanvasTableRenderer{
		canvas:       canvas,
		ctx:          ctx,
		rows:         make([][]string, 0),
		columns:      make([]string, 0),
		rowHeight:    24,
		headerHeight: 32,
		scrollY:      0,
		scrollX:      0,
	}, nil
}

// LoadQueryResults executes a SQL query and loads results into memory
func (r *CanvasTableRenderer) LoadQueryResults(db *sql.DB, query string) error {
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	// Get column names
	r.columns, err = rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %w", err)
	}

	// Initialize column widths based on header text
	r.colWidths = make([]int, len(r.columns))
	for i, col := range r.columns {
		r.colWidths[i] = r.measureText(col) + 20 // padding
	}

	// Scan all rows into memory
	r.rows = make([][]string, 0, 1000)
	values := make([]interface{}, len(r.columns))
	valuePtrs := make([]interface{}, len(r.columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		strValues := make([]string, len(r.columns))
		for i, val := range values {
			if val == nil {
				strValues[i] = "NULL"
			} else {
				strValues[i] = fmt.Sprintf("%v", val)
			}

			// Update column width if needed
			textWidth := r.measureText(strValues[i]) + 20
			if textWidth > r.colWidths[i] {
				r.colWidths[i] = textWidth
			}
		}

		r.rows = append(r.rows, strValues)
	}

	return nil
}

// Render draws the table to the canvas with virtual scrolling
func (r *CanvasTableRenderer) Render() {
	// Clear canvas
	r.ctx.Set("fillStyle", "#ffffff")
	r.ctx.Call("fillRect", 0, 0, r.canvasWidth, r.canvasHeight)

	// Draw header (sticky)
	r.drawHeader()

	// Draw visible rows only
	r.drawRows()

	// Draw scrollbar
	r.drawScrollbar()
}

// drawHeader renders the sticky header row
func (r *CanvasTableRenderer) drawHeader() {
	// Header background
	r.ctx.Set("fillStyle", "#f0f0f0")
	r.ctx.Call("fillRect", 0, 0, r.canvasWidth, r.headerHeight)

	// Header border
	r.ctx.Set("strokeStyle", "#cccccc")
	r.ctx.Call("strokeRect", 0, 0, r.canvasWidth, r.headerHeight)

	// Header text
	r.ctx.Set("fillStyle", "#000000")
	r.ctx.Set("font", "14px sans-serif")
	r.ctx.Set("textBaseline", "middle")

	x := 10 - r.scrollX
	for i, col := range r.columns {
		if x > r.canvasWidth {
			break // Off-screen to the right
		}
		if x+r.colWidths[i] > 0 { // At least partially visible
			r.ctx.Call("fillText", col, x, r.headerHeight/2)
		}
		x += r.colWidths[i]
	}
}

// drawRows renders visible rows using virtual scrolling
func (r *CanvasTableRenderer) drawRows() {
	r.ctx.Set("font", "12px sans-serif")
	r.ctx.Set("textBaseline", "middle")

	// Calculate visible range
	visibleRowStart := r.scrollY / r.rowHeight
	rowsInView := (r.canvasHeight - r.headerHeight) / r.rowHeight
	visibleRowEnd := visibleRowStart + rowsInView + 2 // +2 for buffer

	y := r.headerHeight

	for rowIdx := visibleRowStart; rowIdx < visibleRowEnd && rowIdx < len(r.rows); rowIdx++ {
		row := r.rows[rowIdx]

		// Alternate row background
		if rowIdx%2 == 0 {
			r.ctx.Set("fillStyle", "#fafafa")
		} else {
			r.ctx.Set("fillStyle", "#ffffff")
		}
		r.ctx.Call("fillRect", 0, y, r.canvasWidth, r.rowHeight)

		// Draw row border
		r.ctx.Set("strokeStyle", "#eeeeee")
		r.ctx.Call("strokeRect", 0, y, r.canvasWidth, r.rowHeight)

		// Draw cell values
		r.ctx.Set("fillStyle", "#000000")
		x := 10 - r.scrollX
		for i, cell := range row {
			if x > r.canvasWidth {
				break
			}
			if x+r.colWidths[i] > 0 {
				r.ctx.Call("fillText", cell, x, y+r.rowHeight/2)
			}
			x += r.colWidths[i]
		}

		y += r.rowHeight
	}
}

// drawScrollbar renders a vertical scrollbar
func (r *CanvasTableRenderer) drawScrollbar() {
	if len(r.rows) == 0 {
		return
	}

	scrollbarWidth := 10
	scrollbarX := r.canvasWidth - scrollbarWidth

	totalContentHeight := len(r.rows) * r.rowHeight
	visibleRatio := float64(r.canvasHeight) / float64(totalContentHeight)

	if visibleRatio >= 1.0 {
		return // No scrollbar needed
	}

	scrollbarHeight := int(float64(r.canvasHeight) * visibleRatio)
	scrollbarY := int(float64(r.scrollY) / float64(totalContentHeight) * float64(r.canvasHeight))

	// Draw scrollbar track
	r.ctx.Set("fillStyle", "#f0f0f0")
	r.ctx.Call("fillRect", scrollbarX, 0, scrollbarWidth, r.canvasHeight)

	// Draw scrollbar thumb
	r.ctx.Set("fillStyle", "#888888")
	r.ctx.Call("fillRect", scrollbarX, scrollbarY, scrollbarWidth, scrollbarHeight)
}

// measureText measures the width of text (approximate)
func (r *CanvasTableRenderer) measureText(text string) int {
	// Use canvas measureText for accuracy
	r.ctx.Set("font", "12px sans-serif")
	metrics := r.ctx.Call("measureText", text)
	width := metrics.Get("width").Int()
	return width
}

// SetCanvasSize updates the canvas dimensions
func (r *CanvasTableRenderer) SetCanvasSize(width, height int) {
	r.canvasWidth = width
	r.canvasHeight = height
}

// ScrollTo sets the scroll position
func (r *CanvasTableRenderer) ScrollTo(x, y int) {
	totalContentHeight := len(r.rows) * r.rowHeight
	maxScrollY := totalContentHeight - r.canvasHeight + r.headerHeight

	if y < 0 {
		y = 0
	}
	if y > maxScrollY {
		y = maxScrollY
	}

	r.scrollY = y

	// Horizontal scroll
	totalContentWidth := 0
	for _, w := range r.colWidths {
		totalContentWidth += w
	}
	maxScrollX := totalContentWidth - r.canvasWidth + 20

	if x < 0 {
		x = 0
	}
	if x > maxScrollX {
		x = maxScrollX
	}

	r.scrollX = x
}

// GetRowCount returns the total number of rows
func (r *CanvasTableRenderer) GetRowCount() int {
	return len(r.rows)
}

// ExportJSAPI exports the renderer to JavaScript
func (r *CanvasTableRenderer) ExportJSAPI() js.Value {
	return js.ValueOf(map[string]interface{}{
		"render": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			r.Render()
			return nil
		}),
		"scrollTo": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) >= 2 {
				x := args[0].Int()
				y := args[1].Int()
				r.ScrollTo(x, y)
				r.Render()
			}
			return nil
		}),
		"setSize": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) >= 2 {
				width := args[0].Int()
				height := args[1].Int()
				r.SetCanvasSize(width, height)
				r.Render()
			}
			return nil
		}),
		"getRowCount": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return r.GetRowCount()
		}),
	})
}
