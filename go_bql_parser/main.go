package main

import (
	"encoding/json"
	"fmt"

	bqlparser "bql-parser/internal/wazbean/bql-parser/bql-parser"
)

func init() {
	bqlparser.Exports.ParseBqlToJSON = ParseBQLToJSON
	bqlparser.Exports.ExecuteBql = ExecuteBQL
}

func main() {}

func Parse(query string) (*Query, error) {
	lexer := NewBQLLexer(query)
	if yyParse(lexer) != 0 || lexer.err != nil {
		if lexer.err != nil {
			return nil, lexer.err
		}
		return nil, fmt.Errorf("syntax error")
	}
	return lexer.result, nil
}

func ParseBQLToJSON(query string) string {
	ast, err := Parse(query)
	if err != nil {
		return fmt.Sprintf(`{"error": "%v"}`, err)
	}

	jsonResult, err := json.Marshal(ast)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to serialize AST to JSON: %v"}`, err)
	}

	return string(jsonResult)
}

func ExecuteBQL(query string, ledgerText string) string {
	ast, err := Parse(query)
	if err != nil {
		return fmt.Sprintf(`{"error": "parse error: %v"}`, err)
	}

	ledger, err := ParseLedger(ledgerText)
	if err != nil {
		return fmt.Sprintf(`{"error": "ledger error: %v"}`, err)
	}

	result, err := Execute(ast, ledger)
	if err != nil {
		return fmt.Sprintf(`{"error": "execution error: %v"}`, err)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"error": "serialization error: %v"}`, err)
	}

	return string(jsonResult)
}