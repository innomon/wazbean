# Agent Instructions: Go BQL Parser & Query Engine WASM Component

## 1. Project Overview

This project provides a WASI-compliant WebAssembly component that parses Beancount Query Language (BQL) strings and executes them against Beancount ledger data. The parser is written in Go using `goyacc` and compiled to WASM via TinyGo. The component is designed to be deployed as a [Wassette](https://github.com/microsoft/wassette) component, exposing the parser and query engine as MCP tools for AI agents.

**Deliverables:**
- `go_bql_parser/bql_parser.wasm` — exports `ParseBQLToJSON(query string) string` and `ExecuteBQL(query string, ledgerText string) string`.

**Status:** All tests pass. WASM artifact compiled and ready.

## 2. Project Structure

All parser source lives in `go_bql_parser/`:

| File | Purpose |
|---|---|
| `ast.go` | AST struct definitions (`Query`, `Expression`, `OrderBy`) |
| `bql.y` | `goyacc` grammar — the canonical BQL syntax definition |
| `y.go` | Generated parser (do NOT edit manually) |
| `lexer.go` | Lexer using `text/scanner`; handles keywords, identifiers, single-quoted strings |
| `ledger.go` | Beancount ledger file parser (`ParseLedger`); extracts `Transaction` and `Posting` structs |
| `executor.go` | Query execution engine (`Execute`); handles filtering, projection, GROUP BY, aggregates, ORDER BY |
| `main.go` | `Parse()`, `ParseBQLToJSON()`, and `ExecuteBQL()` entry points |
| `parser_test.go` | Unit tests for BQL parsing (valid and invalid queries) |
| `executor_test.go` | Unit tests for ledger parsing, query execution, aggregation, and end-to-end `ExecuteBQL` |
| `testdata/sample.beancount` | Sample Beancount ledger for testing |
| `wit/world.wit` | WIT world definition for WASI Preview 2 component model |
| `wit/deps/` | WASI WIT dependencies (fetched by `wkg wit fetch`, do NOT edit) |
| `internal/` | Generated Go bindings from `wit-bindgen-go` (do NOT edit) |
| `bql-parser.wasm` | Bundled WIT package (from `wkg wit build`, not the final artifact) |
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

# Fetch WASI WIT dependencies (required once, or after wit changes)
wkg wit fetch --wit-dir ./wit

# Bundle WIT package (required once, or after wit changes)
wkg wit build --wit-dir ./wit -o bql-parser.wasm

# Generate Go bindings from WIT (required once, or after wit changes)
go tool wit-bindgen-go generate --world bql-parser --out internal ./bql-parser.wasm

# Build WASM (WASI Preview 2 — required for Wassette deployment)
tinygo build -o bql_parser.wasm -target wasip2 --wit-package ./wit --wit-world bql-parser .
```

## 4. Development Workflow

1. Edit `bql.y` (grammar), `lexer.go` (tokenizer), `ast.go` (AST structs), `ledger.go` (ledger parser), or `executor.go` (query engine) as needed.
2. **Always** regenerate the parser after grammar changes: `goyacc -o y.go bql.y`
3. Run `go test .` to verify correctness.
4. Rebuild WASM with TinyGo after all tests pass.

**Never edit `y.go` directly** — it is generated from `bql.y`.

**Never edit files in `internal/` or `wit/deps/` directly** — they are generated. After WIT changes, you must re-run `wkg wit fetch`, `wkg wit build`, and `go tool wit-bindgen-go generate` before rebuilding.

## 5. Architecture Notes

### Parser
- The lexer converts BQL keywords to uppercase for case-insensitive matching.
- The `where_clause_opt` uses a struct union type to capture both the field name and the value expression.
- The grammar supports function call syntax: `SUM(amount)`, `COUNT(*)`.
- The `Where` field on `Query` uses `json:"where"` (no `omitempty`) so it always serializes, even when empty.
- The `WhereField` captures the left-hand identifier in `WHERE field = 'value'`.
- The lexer detects unclosed string literals and returns an error via `BQLLexer.err`.
- `Parse()` checks both `yyParse` return value and `lexer.err` for error reporting.

### Ledger Parser
- `ParseLedger(text)` reads line-by-line, recognizing transaction headers (`YYYY-MM-DD * "payee" "narration"`) and indented posting lines.
- Non-transaction directives (`open`, `close`, `balance`, `pad`, `option`) are silently skipped.
- Inline comments (`;`) are stripped, respecting quoted strings.
- Amounts are stored as `float64`; postings without amounts have `HasAmount: false`.

### Query Execution Engine
- Operates on a **posting-row model**: one row per posting, with access to parent transaction fields.
- **FROM** filters at the transaction level by account prefix (keeps all postings from matching transactions).
- **WHERE** filters at the posting level by field equality.
- **GROUP BY** groups rows by key tuple; aggregate functions (`SUM`, `COUNT`) are evaluated per group.
- **ORDER BY** sorts final result rows; supports numeric and string comparison.
- Results are returned as `{"columns": [...], "rows": [[...], ...]}`.

### Component Model Wiring
- Exports are registered in `init()` via generated bindings in `internal/wazbean/bql-parser/bql-parser/`.
- The `bqlparser.Exports.ParseBqlToJSON` and `bqlparser.Exports.ExecuteBql` fields are set to the implementation functions.
- The `//export` annotations are NOT used for wasip2 builds.

## 6. Supported BQL Syntax

```
SELECT expr [, expr ...]
[FROM 'account-prefix']
[WHERE field = 'value']
[GROUP BY expr [, expr ...]]
[ORDER BY expr [ASC|DESC] [, expr [ASC|DESC] ...]]
```

Expressions can be identifiers (`account`, `date`, `amount`, `payee`, `narration`, `currency`, `position`, `flag`) or function calls (`SUM(amount)`, `COUNT(*)`).

## 7. Available Query Fields

| Field | Source | Type | Description |
|---|---|---|---|
| `account` | Posting | string | Account name |
| `amount` | Posting | number | Posting amount |
| `currency` | Posting | string | Currency code |
| `position` | Posting | string | Formatted `amount currency` |
| `date` | Transaction | string | `YYYY-MM-DD` |
| `payee` | Transaction | string | Payee name |
| `narration` | Transaction | string | Description |
| `flag` | Transaction | string | `*` or `!` |

## 8. Wassette Deployment

To deploy this component as a Wassette MCP tool:

1. **Create a WIT interface** in `go_bql_parser/wit/world.wit` defining the exported `parse-bql-to-json` and `execute-bql` functions.
2. **Fetch WIT dependencies and generate bindings**: run `wkg wit fetch --wit-dir ./wit`, then `wkg wit build --wit-dir ./wit -o bql-parser.wasm`, then `go tool wit-bindgen-go generate --world bql-parser --out internal ./bql-parser.wasm`.
3. **Wire the exports** in an `init()` function using plain function assignment on the generated `Exports` struct (e.g., `bqlparser.Exports.ParseBqlToJSON = ...`).
4. **Build for WASI Preview 2** using `tinygo build -o bql_parser.wasm -target wasip2 --wit-package ./wit --wit-world bql-parser .`.
5. **Run with Wassette** using `wassette run ./bql_parser.wasm --allow-stdio`.
6. **Register Wassette as an MCP server** in your AI agent (VS Code, Cursor, Claude Code, Gemini CLI).
7. **Optionally publish** to an OCI registry with `wkg oci push` for remote agent loading.

The BQL parser is a pure computation component — it needs no filesystem, network, or environment access. Only `--allow-stdio` is required.

See `README.md` for full step-by-step deployment instructions.
