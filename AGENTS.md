# Agent Instructions: Go BQL Parser WASM Component

## 1. Project Overview

This project provides a WASI-compliant WebAssembly component that parses Beancount Query Language (BQL) strings into JSON ASTs. The parser is written in Go using `goyacc` and compiled to WASM via TinyGo. The component is designed to be deployed as a [Wassette](https://github.com/microsoft/wassette) component, exposing the parser as an MCP tool for AI agents.

**Deliverable:** `go_bql_parser/bql_parser.wasm` — exports `ParseBQLToJSON(query string) string`.

**Status:** All tests pass. WASM artifact compiled and ready.

## 2. Project Structure

All parser source lives in `go_bql_parser/`:

| File | Purpose |
|---|---|
| `ast.go` | AST struct definitions (`Query`, `Expression`, `OrderBy`) |
| `bql.y` | `goyacc` grammar — the canonical BQL syntax definition |
| `y.go` | Generated parser (do NOT edit manually) |
| `lexer.go` | Lexer using `text/scanner`; handles keywords, identifiers, single-quoted strings |
| `main.go` | `Parse()` and `ParseBQLToJSON()` entry points |
| `main_wasm.go` | JS/WASM bridge (`go:build js && wasm`) |
| `parser_test.go` | Unit tests for valid and invalid queries |
| `bql_parser.wasm` | Compiled WASI artifact |

## 3. Key Commands

```bash
cd go_bql_parser

# Regenerate parser after editing bql.y (REQUIRED)
goyacc -o y.go bql.y

# Run tests
go test .

# Run tests verbose
go test -v .

# Build WASM (WASI Preview 1 — browser/standalone use)
tinygo build -o bql_parser.wasm -target wasi .

# Build WASM (WASI Preview 2 — required for Wassette deployment)
tinygo build -o bql_parser.wasm -target wasip2 --wit-package ./wit --wit-world bql-parser main.go
```

## 4. Development Workflow

1. Edit `bql.y` (grammar), `lexer.go` (tokenizer), or `ast.go` (AST structs) as needed.
2. **Always** regenerate the parser after grammar changes: `goyacc -o y.go bql.y`
3. Run `go test .` to verify correctness.
4. Rebuild WASM with TinyGo after all tests pass.

**Never edit `y.go` directly** — it is generated from `bql.y`.

## 5. Architecture Notes

- The lexer converts BQL keywords to uppercase for case-insensitive matching.
- The `where_clause_opt` empty production must explicitly assign `$$ = Expression{}` to avoid stale stack values from `goyacc`.
- The `Where` field on `Query` uses `json:"where"` (no `omitempty`) so it always serializes, even when empty.
- The lexer detects unclosed string literals and returns an error via `BQLLexer.err`.
- `Parse()` checks both `yyParse` return value and `lexer.err` for error reporting.

## 6. Supported BQL Syntax

```
SELECT expr [, expr ...]
[FROM 'source']
[WHERE ident = 'value']
[GROUP BY expr [, expr ...]]
[ORDER BY expr [ASC|DESC] [, expr [ASC|DESC] ...]]
```

## 7. Wassette Deployment

To deploy this component as a Wassette MCP tool:

1. **Create a WIT interface** in `go_bql_parser/wit/world.wit` defining the exported `parse-bql-to-json` function.
2. **Generate Go bindings** with `wit-bindgen-go`.
3. **Wire the export** in an `init()` function using the `cm.Result` type from `go.bytecodealliance.org/cm`.
4. **Build for WASI Preview 2** using `tinygo build -target wasip2 --wit-package ./wit --wit-world bql-parser`.
5. **Run with Wassette** using `wassette run ./bql_parser.wasm --allow-stdio`.
6. **Register Wassette as an MCP server** in your AI agent (VS Code, Cursor, Claude Code, Gemini CLI).
7. **Optionally publish** to an OCI registry with `wkg oci push` for remote agent loading.

The BQL parser is a pure computation component — it needs no filesystem, network, or environment access. Only `--allow-stdio` is required.

See `README.md` for full step-by-step deployment instructions.
