package sqliter

import (
	"fmt"
	"io"
)

// --- Fallback implementations ---

func fmt_StartHTMLTable(w io.Writer, headers []string) {
	io.WriteString(w, "<!DOCTYPE html><html><head><title>Data</title></head><body><table border='1'><thead><tr>")
	for _, h := range headers {
		io.WriteString(w, "<th>"+h+"</th>")
	}
	io.WriteString(w, "</tr></thead><tbody>")
	flush(w)
}

func fmt_WriteHTMLRow(w io.Writer, index int, cells []string) {
	io.WriteString(w, "<tr>")
	// Fallback doesn't necessarily need the index column since it's "simple",
	// but to match the structure we should probably add it or ignore it.
	// Let's add it for consistency.
	io.WriteString(w, "<td>"+fmt.Sprint(index)+"</td>")
	for _, c := range cells {
		io.WriteString(w, "<td>"+c+"</td>")
	}
	io.WriteString(w, "</tr>")
	flush(w)
}

func fmt_EndHTMLTable(w io.Writer) {
	io.WriteString(w, "</tbody></table></body></html>")
	flush(w)
}
