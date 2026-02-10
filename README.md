# Wazbean: BQL Parser WASM Component

A self-contained WebAssembly (WASM) component that parses [Beancount Query Language (BQL)](https://beancount.github.io/docs/beancount_query_language.html) strings into JSON ASTs. Designed to be deployed as a [Wassette](https://github.com/microsoft/wassette) component, exposing the parser as an MCP tool for AI agents.

## Overview

Wazbean provides a BQL parser compiled to a WASI-compliant `.wasm` module. The parser is written in Go using `goyacc` for grammar-driven parsing and compiled to WASM via TinyGo. When deployed as a Wassette component, the parser becomes an MCP tool that AI agents (Claude, GitHub Copilot, Cursor, Gemini CLI) can invoke to parse BQL queries.

## Project Structure

```
go_bql_parser/
├── ast.go              # AST struct definitions (Query, Expression, OrderBy)
├── bql.y               # goyacc grammar — canonical BQL syntax definition
├── y.go                # Generated parser (do NOT edit manually)
├── lexer.go            # Lexer using Go's text/scanner
├── main.go             # Parse() and ParseBQLToJSON() entry points
├── parser_test.go      # Unit tests
└── bql_parser.wasm     # Compiled WASI artifact
```

## Exported Function

```
ParseBQLToJSON(query string) string
```

Accepts a BQL query string, returns a JSON-serialized AST or an error object.

### Example

**Input:** `SELECT account, balance FROM 'Expenses:Cash' WHERE category = 'Groceries' ORDER BY balance DESC`

**Output:**
```json
{
  "select": [{"literal": "account"}, {"literal": "balance"}],
  "from": "Expenses:Cash",
  "where": {"literal": "Groceries"},
  "group_by": null,
  "order_by": [{"expression": {"literal": "balance"}, "ascending": false}]
}
```

## Supported BQL Clauses

- `SELECT` — comma-separated list of identifiers
- `FROM` — single-quoted source string
- `WHERE` — condition in the form `identifier = 'value'`
- `GROUP BY` — comma-separated list of identifiers
- `ORDER BY` — comma-separated list of identifiers with optional `ASC`/`DESC`

## Build

### Prerequisites

- Go 1.25+
- `goyacc` (bundled with `golang.org/x/tools`)
- TinyGo (for WASM compilation)

### Commands

```bash
cd go_bql_parser

# Regenerate parser from grammar (required after editing bql.y)
goyacc -o y.go bql.y

# Run tests
go test .

# Build WASM artifact (WASI Preview 1 — browser/standalone use)
tinygo build -o bql_parser.wasm -target wasi .
```

## Deploying as a Wassette Component

Wassette is Microsoft's open-source runtime that runs WebAssembly Components as secure, sandboxed MCP tools for AI agents. Deploying the BQL parser as a Wassette component makes it available to any MCP-compatible AI agent.

### Step 1: Create a WIT Interface

Wassette uses the [WebAssembly Component Model](https://component-model.bytecodealliance.org/) and requires a [WIT (WebAssembly Interface Types)](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md) file to define the component's exported interface.

Create `go_bql_parser/wit/world.wit`:

```wit
package local:bql-parser;

interface bql {
    /// Parse a BQL query string into a JSON AST.
    /// Returns a JSON string containing the parsed AST on success,
    /// or a JSON object with an "error" field on failure.
    parse-bql-to-json: func(query: string) -> result<string, string>;
}

world bql-parser {
    include wasi:cli/imports@0.2.0;

    export bql;
}
```

### Step 2: Generate Go Bindings

Use the `wit-bindgen-go` tool to generate Go bindings from the WIT definition:

```bash
cd go_bql_parser
go run go.bytecodealliance.org/cmd/wit-bindgen-go@v0.6.2 generate -o gen ./wit
```

This creates a `gen/` directory with typed Go bindings that map the WIT interface to Go function signatures.

### Step 3: Wire Up the Exported Function

Create or update an init function in your Go source to register `ParseBQLToJSON` with the generated bindings:

```go
package main

import (
    "go_bql_parser/gen/local/bql-parser/bql"
    "go.bytecodealliance.org/cm"
)

func init() {
    bql.Exports.ParseBqlToJson = parseBqlToJson
}

type ParseResult = cm.Result[string, string, string]

func parseBqlToJson(query string) ParseResult {
    ast, err := Parse(query)
    if err != nil {
        return cm.Err[ParseResult](err.Error())
    }
    jsonResult, err := json.Marshal(ast)
    if err != nil {
        return cm.Err[ParseResult]("Failed to serialize AST: " + err.Error())
    }
    return cm.OK[ParseResult](string(jsonResult))
}
```

### Step 4: Add Dependencies

Update `go.mod` to include the Component Model runtime library:

```bash
go get go.bytecodealliance.org/cm@v0.2.2
```

### Step 5: Build for WASI Preview 2

Wassette requires components built for WASI Preview 2 (`wasip2`):

```bash
tinygo build -o bql_parser.wasm -target wasip2 \
    --wit-package ./wit --wit-world bql-parser main.go
```

### Step 6: (Optional) Inject WIT Documentation

Embed the WIT interface documentation into the component binary so AI agents can discover what the tool does:

```bash
# From repository root (requires wassette CLI)
wassette inject-docs go_bql_parser/bql_parser.wasm go_bql_parser/wit
```

### Step 7: Install Wassette

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/microsoft/wassette/main/install.sh | bash

# macOS (Homebrew)
brew tap microsoft/wassette https://github.com/microsoft/wassette
brew install wassette

# Verify
wassette --version
```

### Step 8: Run Locally with Wassette

```bash
wassette run ./go_bql_parser/bql_parser.wasm --allow-stdio
```

### Step 9: Configure as an MCP Server

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

### Step 10: (Optional) Publish to an OCI Registry

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

The initial approach of wrapping the official Beancount C++ parser was abandoned due to a broken Bazel build environment on the `cpp` branch. The current implementation is a ground-up rewrite in Go.
