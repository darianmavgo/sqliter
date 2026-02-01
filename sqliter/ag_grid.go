package sqliter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/darianmavgo/banquet/sqlite"
)

type AgFilter struct {
	FilterType string      `json:"filterType"`
	Type       string      `json:"type"`
	Filter     interface{} `json:"filter"`   // Can be string or number
	FilterTo   interface{} `json:"filterTo"` // For inRange

	// For complex filters
	Operator   string    `json:"operator"`
	Condition1 *AgFilter `json:"condition1"`
	Condition2 *AgFilter `json:"condition2"`
}

type AgFilterModel map[string]AgFilter

func BuildWhereClause(filterModelJSON string) (string, error) {
	if filterModelJSON == "" {
		return "", nil
	}

	var model AgFilterModel
	if err := json.Unmarshal([]byte(filterModelJSON), &model); err != nil {
		return "", err
	}

	var conditions []string
	for col, filter := range model {
		cond, err := buildCondition(col, filter)
		if err != nil {
			return "", err
		}
		if cond != "" {
			conditions = append(conditions, "("+cond+")")
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " AND "), nil
}

func buildCondition(col string, filter AgFilter) (string, error) {
	// Handle complex conditions (AND/OR)
	if filter.Operator != "" {
		if filter.Condition1 == nil || filter.Condition2 == nil {
			return "", fmt.Errorf("invalid complex filter for column %s", col)
		}
		c1, err := buildCondition(col, *filter.Condition1)
		if err != nil {
			return "", err
		}
		c2, err := buildCondition(col, *filter.Condition2)
		if err != nil {
			return "", err
		}
		op := "AND"
		if strings.ToUpper(filter.Operator) == "OR" {
			op = "OR"
		}
		return fmt.Sprintf("(%s %s %s)", c1, op, c2), nil
	}

	colEscaped := sqlite.QuoteIdentifier(col)

	// Check filterType
	if filter.FilterType == "text" {
		val := fmt.Sprintf("%v", filter.Filter)
		valEscaped := strings.ReplaceAll(val, "'", "''")

		switch filter.Type {
		case "equals":
			return fmt.Sprintf("%s = '%s'", colEscaped, valEscaped), nil
		case "notEqual":
			return fmt.Sprintf("%s != '%s'", colEscaped, valEscaped), nil
		case "contains":
			return fmt.Sprintf("%s LIKE '%%%s%%'", colEscaped, valEscaped), nil
		case "notContains":
			return fmt.Sprintf("%s NOT LIKE '%%%s%%'", colEscaped, valEscaped), nil
		case "startsWith":
			return fmt.Sprintf("%s LIKE '%s%%'", colEscaped, valEscaped), nil
		case "endsWith":
			return fmt.Sprintf("%s LIKE '%%%s'", colEscaped, valEscaped), nil
		default:
			return "", fmt.Errorf("unsupported text filter type: %s", filter.Type)
		}
	} else if filter.FilterType == "number" {
		val, err := validateNumber(filter.Filter)
		if err != nil {
			return "", err
		}

		switch filter.Type {
		case "equals":
			return fmt.Sprintf("%s = %s", colEscaped, val), nil
		case "notEqual":
			return fmt.Sprintf("%s != %s", colEscaped, val), nil
		case "greaterThan":
			return fmt.Sprintf("%s > %s", colEscaped, val), nil
		case "greaterThanOrEqual":
			return fmt.Sprintf("%s >= %s", colEscaped, val), nil
		case "lessThan":
			return fmt.Sprintf("%s < %s", colEscaped, val), nil
		case "lessThanOrEqual":
			return fmt.Sprintf("%s <= %s", colEscaped, val), nil
		case "inRange":
			valTo, err := validateNumber(filter.FilterTo)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s >= %s AND %s <= %s", colEscaped, val, colEscaped, valTo), nil
		default:
			return "", fmt.Errorf("unsupported number filter type: %s", filter.Type)
		}
	}

	return "", nil // Ignore other types (date, set) for now
}

func validateNumber(v interface{}) (string, error) {
	switch val := v.(type) {
	case float64:
		// Format without scientific notation if possible, or just standard %v
		// %f might be safer.
		return fmt.Sprintf("%f", val), nil
	case int:
		return fmt.Sprintf("%d", val), nil
	case string:
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return "", fmt.Errorf("invalid number value: %s", val)
		}
		return val, nil
	default:
		return "", fmt.Errorf("invalid number value type: %T", v)
	}
}
