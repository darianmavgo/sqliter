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

// ConstructInsert builds an INSERT statement.
// It returns the SQL string and the list of values.
func ConstructInsert(table string, data map[string]interface{}) (string, []interface{}) {
	cols := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data))
	placeholders := make([]string, 0, len(data))

	for k, v := range data {
		cols = append(cols, k)
		vals = append(vals, v)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	return query, vals
}

// ConstructUpdate builds an UPDATE statement.
// It returns the SQL string and the list of values (update values + where values).
func ConstructUpdate(table string, data map[string]interface{}, where map[string]interface{}) (string, []interface{}) {
	setClauses := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data)+len(where))

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		vals = append(vals, v)
	}

	whereClauses := make([]string, 0, len(where))
	for k, v := range where {
		if v == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", k))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", k))
			vals = append(vals, v)
		}
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, strings.Join(setClauses, ", "), strings.Join(whereClauses, " AND "))
	return query, vals
}

// ConstructDelete builds a DELETE statement.
// It returns the SQL string and the list of values.
func ConstructDelete(table string, where map[string]interface{}) (string, []interface{}) {
	whereClauses := make([]string, 0, len(where))
	vals := make([]interface{}, 0, len(where))

	for k, v := range where {
		if v == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", k))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", k))
			vals = append(vals, v)
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, strings.Join(whereClauses, " AND "))
	return query, vals
}
