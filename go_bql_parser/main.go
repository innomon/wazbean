package main

import (
	"encoding/json"
	"fmt"
)

// main is required by the Go compiler, but it does nothing in a WASI library.
func main() {}

// Parse takes a BQL query string and returns the parsed AST.
// This function is the core logic of our parser.
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

// ParseBQLToJSON is the function that will be exported from the WASM module.
// It takes a BQL query string and returns a JSON string.
// The //export comment is a TinyGo directive.
//export ParseBQLToJSON
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