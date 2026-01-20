package common

import (
	"fmt"
	"strings"

	"github.com/darianmavgo/banquet"
)

// ConstructSQL builds a SQL query string from a Banquet struct.
// WARNING: This implementation concatenates strings directly and is vulnerable to SQL injection.
// It is intended for demonstration purposes or strictly trusted internal inputs.
func ConstructSQL(bq *banquet.Banquet) string {
	selectClause := "*"
	if len(bq.Select) > 0 {
		cols := []string{}
		for _, s := range bq.Select {
			if s == "*" {
				cols = append(cols, "*")
			} else if s != bq.Table {
				cols = append(cols, s)
			}
		}
		if len(cols) > 0 {
			selectClause = strings.Join(cols, ", ")
		}
	}

    if selectClause == "" {
        selectClause = "*"
    }

	q := fmt.Sprintf("SELECT %s FROM %s", selectClause, bq.Table)

	if bq.Where != "" {
		q += " WHERE " + bq.Where
	}

    if bq.GroupBy != "" {
        q += " GROUP BY " + bq.GroupBy
    }

    if bq.Having != "" {
        q += " HAVING " + bq.Having
    }

    if bq.OrderBy != "" {
        q += " ORDER BY " + bq.OrderBy
        if bq.SortDirection != "" {
            q += " " + bq.SortDirection
        }
    }

	if bq.Limit != "" {
		q += " LIMIT " + bq.Limit
	}

	if bq.Offset != "" {
		q += " OFFSET " + bq.Offset
	}

	return q
}
