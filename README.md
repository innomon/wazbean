# Wazbean: BQL Parser & Query Engine WASM Component

A self-contained WebAssembly (WASM) component that parses [Beancount Query Language (BQL)](https://beancount.github.io/docs/beancount_query_language.html) strings and executes them against Beancount ledger data. Designed to be deployed as a [Wassette](https://github.com/microsoft/wassette) component, exposing the parser and query engine as MCP tools for AI agents.

## Overview

Wazbean provides a BQL parser and query execution engine compiled to a WASI-compliant `.wasm` module. The parser is written in Go using `goyacc` for grammar-driven parsing and compiled to WASM via TinyGo. It can:

1. **Parse** BQL query strings into JSON ASTs
2. **Parse** Beancount `.beancount` ledger files into an in-memory transaction/posting model
3. **Execute** BQL queries against ledger data, returning tabular JSON results
4. **Check syntax** of `.beancount` ledger files, returning structured errors with line numbers

When deployed as a Wassette component, these capabilities become MCP tools that AI agents (Claude, GitHub Copilot, Cursor, Gemini CLI) can invoke.

## Project Structure

```
go_bql_parser/
├── ast.go              # AST struct definitions (Query, Expression, OrderBy)
├── bql.y               # goyacc grammar — canonical BQL syntax definition
├── y.go                # Generated parser (do NOT edit manually)
├── lexer.go            # Lexer using Go's text/scanner
├── ledger.go           # Beancount ledger file parser (Transaction, Posting)
├── executor.go         # Query execution engine (filter, project, group, sort)
├── syntax.go           # Beancount ledger syntax checker
├── main.go             # Parse(), ParseBQLToJSON(), ExecuteBQL(), and CheckBeancountSyntax() entry points
├── parser_test.go      # Parser unit tests
├── executor_test.go    # Execution engine unit tests
├── syntax_test.go      # Syntax checker unit tests
├── testdata/
│   └── sample.beancount  # Sample ledger for testing
├── wit/
│   ├── world.wit       # WIT world definition for WASI Preview 2 component
│   └── deps/           # WASI WIT dependencies (fetched by `wkg wit fetch`)
├── internal/           # Generated Go bindings (from `wit-bindgen-go`, do NOT edit)
├── bql-parser.wasm     # Bundled WIT package (from `wkg wit build`)
└── bql_parser.wasm     # Compiled WASI artifact
```

## Exported Functions

### ParseBQLToJSON

```
ParseBQLToJSON(query string) string
```

Accepts a BQL query string, returns a JSON-serialized AST or an error object.

**Input:** `SELECT account, balance FROM 'Expenses:Cash' WHERE category = 'Groceries' ORDER BY balance DESC`

**Output:**
```json
{
  "select": [{"literal": "account"}, {"literal": "balance"}],
  "from": "Expenses:Cash",
  "where": {"literal": "Groceries"},
  "where_field": "category",
  "order_by": [{"expression": {"literal": "balance"}, "ascending": false}]
}
```

### ExecuteBQL

```
ExecuteBQL(query string, ledgerText string) string
```

Accepts a BQL query string and the full text content of a Beancount ledger file. Parses both, executes the query against the ledger data, and returns tabular JSON results.

**Input query:** `SELECT account, SUM(amount) WHERE account = 'Expenses:Food:Groceries' GROUP BY account`

**Output:**
```json
{
  "columns": ["account", "sum(amount)"],
  "rows": [["Expenses:Food:Groceries", 607.65]]
}
```

### CheckBeancountSyntax

```
CheckBeancountSyntax(ledgerText string) string
```

Accepts the full text content of a Beancount ledger file and validates its syntax. Returns a JSON object indicating whether the file is valid, with an array of errors including line numbers and messages.

**Valid input output:**
```json
{
  "valid": true,
  "errors": []
}
```

**Invalid input output:**
```json
{
  "valid": false,
  "errors": [
    {"line": 1, "message": "transaction has no postings"},
    {"line": 5, "message": "unknown directive: foobar"}
  ]
}
```

**Checks performed:**
- Transaction structure: requires `*` or `!` flag and quoted narration
- Transactions must have at least one posting
- Posting syntax validation (account name, optional amount and currency)
- Indented lines must appear inside a transaction
- Known directives: `open`, `close`, `balance`, `pad`, `event`, `note`, `document`, `custom`, `commodity`, `price`, `query`, `plugin`
- Directives that require an account (`open`, `close`, `balance`, `pad`) must have one
- Top-level `option`, `include`, `plugin`, `pushtag`, `poptag` lines are accepted

## Query Execution Model

The engine operates on **posting rows** — one row per posting in the ledger, with access to the parent transaction's fields.

### Available Fields

| Field | Source | Type | Description |
|---|---|---|---|
| `account` | Posting | string | Account name (e.g. `Expenses:Food:Groceries`) |
| `amount` | Posting | number | Posting amount (e.g. `87.34`) |
| `currency` | Posting | string | Currency code (e.g. `USD`) |
| `position` | Posting | string | Formatted amount + currency (e.g. `87.34 USD`) |
| `date` | Transaction | string | Transaction date (`YYYY-MM-DD`) |
| `payee` | Transaction | string | Payee (e.g. `Whole Foods`) |
| `narration` | Transaction | string | Description (e.g. `Weekly groceries`) |
| `flag` | Transaction | string | Transaction flag (`*` or `!`) |

### Filtering

- **`FROM 'prefix'`** — Transaction-level filter. Selects all postings from transactions that have at least one posting whose account starts with the given prefix. This preserves both sides of matching transactions.
- **`WHERE field = 'value'`** — Posting-level filter. Keeps only postings where the specified field exactly matches the value.

### Aggregate Functions

When `GROUP BY` is used (or aggregate functions appear in `SELECT`):

- **`SUM(amount)`** — Sum of the `amount` field across grouped postings
- **`COUNT(*)`** — Number of postings in each group

### Sorting

- **`ORDER BY field [ASC|DESC]`** — Sort results by column value. Works on both plain and grouped queries. Numeric values are compared numerically; strings are compared lexicographically.

## Supported BQL Syntax

```
SELECT expr [, expr ...]
[FROM 'account-prefix']
[WHERE field = 'value']
[GROUP BY expr [, expr ...]]
[ORDER BY expr [ASC|DESC] [, expr [ASC|DESC] ...]]
```

Expressions can be:
- Identifiers: `account`, `date`, `amount`, `payee`, `narration`, `currency`, `position`, `flag`
- Function calls: `SUM(amount)`, `COUNT(*)`

## Beancount Ledger Format

The ledger parser recognizes transaction directives and their postings. All other Beancount directives (`open`, `close`, `balance`, `pad`, `option`, etc.) are silently skipped.

**Transaction format:**
```
YYYY-MM-DD * "Payee" "Narration"
  Account:Name    amount CURRENCY
  Account:Name   -amount CURRENCY
```

The payee string is optional. Postings without an explicit amount are parsed with `has_amount: false`.

## Example Queries

```sql
-- List all expense postings with dates
SELECT date, account, position WHERE account = 'Expenses:Food:Groceries'

-- Total spending by expense category
SELECT account, SUM(amount) FROM 'Expenses' GROUP BY account ORDER BY sum(amount) DESC

-- All salary deposits
SELECT date, amount WHERE account = 'Income:Salary:AcmeCo'

-- Count postings per account
SELECT account, COUNT(*) GROUP BY account ORDER BY count(*) DESC
```

## Build

### Prerequisites

- Go 1.25+
- `goyacc` (bundled with `golang.org/x/tools`)
- TinyGo (for WASM compilation)
- `wkg` (WebAssembly package manager, `cargo install wkg`)
- `wit-bindgen-go` (added as Go tool dependency in go.mod)

### Commands

```bash
cd go_bql_parser

# Regenerate parser from grammar (required after editing bql.y)
goyacc -o y.go bql.y

# Run tests
go test .

# Build WASM artifact (WASI Preview 1 — browser/standalone use)
tinygo build -o bql_parser.wasm -target wasi .

# Fetch WASI WIT dependencies (required once, or after wit changes)
wkg wit fetch --wit-dir ./wit

# Bundle WIT package
wkg wit build --wit-dir ./wit -o bql-parser.wasm

# Generate Go bindings from WIT (required once, or after wit changes)
go tool wit-bindgen-go generate --world bql-parser --out internal ./bql-parser.wasm

# Build WASM (WASI Preview 2 — required for Wassette deployment)
tinygo build -o bql_parser.wasm -target wasip2 --wit-package ./wit --wit-world bql-parser .
```

## Deploying as a Wassette Component

Wassette is Microsoft's open-source runtime that runs WebAssembly Components as secure, sandboxed MCP tools for AI agents. Deploying the BQL parser as a Wassette component makes it available to any MCP-compatible AI agent.

### Step 1: Create a WIT Interface

Wassette uses the [WebAssembly Component Model](https://component-model.bytecodealliance.org/) and requires a [WIT (WebAssembly Interface Types)](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md) file to define the component's exported interface.

Create `go_bql_parser/wit/world.wit`:

```wit
package wazbean:bql-parser;

world bql-parser {
    include wasi:cli/imports@0.2.0;

    export parse-bql-to-json: func(query: string) -> string;
    export execute-bql: func(query: string, ledger-text: string) -> string;
    export check-beancount-syntax: func(ledger-text: string) -> string;
}
```

### Step 2: Fetch WASI WIT Dependencies

Use `wkg` to fetch the WASI WIT dependencies referenced by the world:

```bash
cd go_bql_parser
wkg wit fetch --wit-dir ./wit
```

This populates `wit/deps/` with the required WASI interface definitions.

### Step 3: Bundle WIT Package and Generate Go Bindings

Bundle the WIT package, then use `wit-bindgen-go` (added as a Go tool dependency in `go.mod`) to generate typed Go bindings:

```bash
# Bundle WIT package
wkg wit build --wit-dir ./wit -o bql-parser.wasm

# Generate Go bindings into internal/ directory
go tool wit-bindgen-go generate --world bql-parser --out internal ./bql-parser.wasm
```

This creates an `internal/` directory with typed Go bindings that map the WIT exports to Go function signatures.

### Step 4: Wire Up the Exported Functions

Create or update an init function in your Go source to register both exports with the generated bindings:

```go
package main

import (
	bqlparser "bql-parser/internal/wazbean/bql-parser/bql-parser"
)

func init() {
	bqlparser.Exports.ParseBqlToJSON = ParseBQLToJSON
	bqlparser.Exports.ExecuteBql = ExecuteBQL
	bqlparser.Exports.CheckBeancountSyntax = CheckBeancountSyntax
}
```

The exported functions return plain strings — the existing `ParseBQLToJSON`, `ExecuteBQL`, and `CheckBeancountSyntax` functions are wired directly.

### Step 5: Build for WASI Preview 2

Wassette requires components built for WASI Preview 2 (`wasip2`):

```bash
tinygo build -o bql_parser.wasm -target wasip2 \
    --wit-package ./wit --wit-world bql-parser .
```

### Step 6: Install Wassette

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/microsoft/wassette/main/install.sh | bash

# macOS (Homebrew)
brew tap microsoft/wassette https://github.com/microsoft/wassette
brew install wassette

# Verify
wassette --version
```

### Step 7: Run Locally with Wassette

```bash
wassette run ./go_bql_parser/bql_parser.wasm --allow-stdio
```

### Step 8: Configure as an MCP Server

Register Wassette with your AI agent so it can discover and invoke the BQL parser tool.

**VS Code (GitHub Copilot):**
```bash
code --add-mcp '{"name":"Wassette","command":"wassette","args":["run"]}'
```

**Cursor** — add to MCP settings:
```json
{
  "mcpServers": {
    "wassette": {
      "command": "wassette",
      "args": ["run"]
    }
  }
}
```

**Claude Code:**
```bash
claude mcp add -- wassette wassette run
```

**Gemini CLI** — add to `~/.gemini/settings.json`:
```json
{
  "mcpServers": {
    "wassette": {
      "command": "wassette",
      "args": ["run"]
    }
  }
}
```

Once configured, tell your agent to load the component:

```
Please load the BQL parser component from ./go_bql_parser/bql_parser.wasm
```

### Step 9: (Optional) Publish to an OCI Registry

Publishing to an OCI registry allows remote loading without distributing the `.wasm` file directly:

```bash
# Install wkg (WebAssembly package manager)
cargo install wkg

# Publish to GitHub Container Registry
wkg oci push ghcr.io/<your-org>/bql-parser:1.0.0 go_bql_parser/bql_parser.wasm
```

Once published, agents can load the component remotely:

```
Please load the BQL parser from oci://ghcr.io/<your-org>/bql-parser:latest
```

### Security Model

Wassette runs all components in a deny-by-default sandbox. The BQL parser is a pure computation component — it requires no filesystem, network, or environment access. Only `--allow-stdio` is needed for input/output. This makes it one of the safest possible Wassette components to deploy.

## Background

The initial approach of wrapping the official Beancount C++ parser (via a C++ WASM wrapper) was abandoned due to a broken Bazel build environment on the `cpp` branch. That code has been removed. The current implementation is a ground-up rewrite in Go.
