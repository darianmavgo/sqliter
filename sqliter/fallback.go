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

func fmt_WriteHTMLRow(w io.Writer, cells []string) {
	io.WriteString(w, "<tr>")
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

func fmt_StartTableList(w io.Writer) {
	io.WriteString(w, "<!DOCTYPE html><html><head><title>Tables</title></head><body><ul>")
	flush(w)
}

func fmt_WriteTableLink(w io.Writer, name, url, kind string) error {
	var err error
	if kind != "" {
		_, err = fmt.Fprintf(w, "<li><a href='%s'>%s</a> (%s)</li>", url, name, kind)
	} else {
		_, err = fmt.Fprintf(w, "<li><a href='%s'>%s</a></li>", url, name)
	}
	return err
}

func fmt_EndTableList(w io.Writer) {
	io.WriteString(w, "</ul></body></html>")
	flush(w)
}
