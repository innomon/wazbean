package main

import (
	"encoding/json"
	"testing"
)

func TestParseValidQueries(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectedJSON string
	}{
		{
			name:         "simple select",
			query:        "SELECT account",
			expectedJSON: `{"select":[{"literal":"account"}],"where":{}}`,
		},
		{
			name:         "select with multiple expressions",
			query:        "SELECT account, balance",
			expectedJSON: `{"select":[{"literal":"account"},{"literal":"balance"}],"where":{}}`,
		},
		{
			name:         "select from where group by order by",
			query:        "SELECT account, balance FROM 'Expenses:Cash' WHERE category = 'Groceries' GROUP BY account ORDER BY balance DESC",
			expectedJSON: `{"select":[{"literal":"account"},{"literal":"balance"}],"from":"Expenses:Cash","where":{"literal":"Groceries"},"group_by":[{"literal":"account"}],"order_by":[{"expression":{"literal":"balance"},"ascending":false}]}`,
		},
		{
			name:         "order by ascending implicit",
			query:        "SELECT account ORDER BY account",
			expectedJSON: `{"select":[{"literal":"account"}],"where":{},"order_by":[{"expression":{"literal":"account"},"ascending":true}]}`,
		},
		{
			name:         "order by explicit ascending",
			query:        "SELECT account ORDER BY account ASC",
			expectedJSON: `{"select":[{"literal":"account"}],"where":{},"order_by":[{"expression":{"literal":"account"},"ascending":true}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.query, err)
			}

			jsonResult, err := json.Marshal(ast)
			if err != nil {
				t.Fatalf("Failed to marshal AST to JSON: %v", err)
			}

			if string(jsonResult) != tt.expectedJSON {
				t.Errorf("Parse(%q) got JSON:\n%s\nwant JSON:\n%s", tt.query, string(jsonResult), tt.expectedJSON)
			}
		})
	}
}

func TestParseInvalidQueries(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "missing select keyword",
			query: "account",
		},
		{
			name:  "invalid token in select list",
			query: "SELECT account, 123 invalid",
		},
		{
			name:  "unclosed string",
			query: "SELECT account FROM 'Expenses:Cash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.query)
			if err == nil {
				t.Errorf("Parse(%q) expected an error, but got none", tt.query)
			}
		})
	}
}