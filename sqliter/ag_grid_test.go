package sqliter

import (
	"strings"
	"testing"
)

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name          string
		filterJSON    string
		expected      string
		expectError   bool
		expectedInErr string
	}{
		{
			name:       "Simple Text Equals",
			filterJSON: `{"name": {"filterType": "text", "type": "equals", "filter": "Alice"}}`,
			expected:   `"name" = 'Alice'`,
		},
		{
			name:       "Simple Text Contains",
			filterJSON: `{"desc": {"filterType": "text", "type": "contains", "filter": "bob"}}`,
			expected:   `"desc" LIKE '%bob%'`,
		},
		{
			name:       "Text with Quotes",
			filterJSON: `{"name": {"filterType": "text", "type": "equals", "filter": "O'Reilly"}}`,
			expected:   `"name" = 'O''Reilly'`,
		},
		{
			name:       "Number Greater Than",
			filterJSON: `{"age": {"filterType": "number", "type": "greaterThan", "filter": 21}}`,
			expected:   `"age" > 21.000000`, // %f adds decimal places usually
		},
		{
			name:       "Number In Range",
			filterJSON: `{"price": {"filterType": "number", "type": "inRange", "filter": 10, "filterTo": 20}}`,
			expected:   `"price" >= 10.000000 AND "price" <= 20.000000`,
		},
		{
			name:       "Complex AND",
			filterJSON: `{"status": {"filterType": "text", "operator": "AND", "condition1": {"filterType": "text", "type": "contains", "filter": "active"}, "condition2": {"filterType": "text", "type": "notEqual", "filter": "inactive_temp"}}}`,
			expected:   `("status" LIKE '%active%' AND "status" != 'inactive_temp')`,
		},
		{
			name:       "Complex OR",
			filterJSON: `{"status": {"filterType": "text", "operator": "OR", "condition1": {"filterType": "text", "type": "equals", "filter": "A"}, "condition2": {"filterType": "text", "type": "equals", "filter": "B"}}}`,
			expected:   `("status" = 'A' OR "status" = 'B')`,
		},
		{
			name:       "Multiple Columns",
			filterJSON: `{"col1": {"filterType": "text", "type": "equals", "filter": "A"}, "col2": {"filterType": "number", "type": "equals", "filter": 1}}`,
			expected:   "CHECK_BOTH",
		},
		{
			name:       "Empty",
			filterJSON: ``,
			expected:   ``,
		},
		{
			name:        "Invalid JSON",
			filterJSON:  `{invalid`,
			expectError: true,
		},
		{
			name:        "Invalid Number Injection",
			filterJSON:  `{"age": {"filterType": "number", "type": "greaterThan", "filter": "1 OR 1=1"}}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildWhereClause(tt.filterJSON)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected == "CHECK_BOTH" {
				// Special check for multiple columns
				// Also update number expectation for %f
				if !strings.Contains(got, `"col1" = 'A'`) || !strings.Contains(got, `"col2" = 1.000000`) || !strings.Contains(got, ` AND `) {
					t.Errorf("Got %s, expected to contain conditions for col1 and col2", got)
				}
			} else {
				if tt.expected == "" {
					if got != "" {
						t.Errorf("Got %q, expected empty string", got)
					}
				} else {
					// We expect parens wrapping the condition
					expectedWithParens := "(" + tt.expected + ")"
					if got != expectedWithParens {
						t.Errorf("Got %q, expected %q", got, expectedWithParens)
					}
				}
			}
		})
	}
}
