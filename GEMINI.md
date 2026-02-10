# Wazbean: Go BQL Parser WASM Component

This document outlines the plan and progress for creating a WebAssembly (WASM) component for parsing the Beancount Query Language (BQL). The project pivoted from an initial C++-based approach to a full rewrite in Go.

## Current Status

**Complete.** The BQL parser is fully functional — all unit tests pass and the WASM artifact (`bql_parser.wasm`, 1.1MB) has been compiled with TinyGo for the WASI target.

---

## Project Log

### 1. Initial C++ Approach (Blocked and Abandoned)

The original plan was to create a WASM wrapper around the official Beancount v3 C++ BQL parser.

*   **Investigation:** The C++ source code was successfully located on the `cpp` branch of the Beancount repository. The project uses the Bazel build system.
*   **Blocker 1: Broken Build System:** The Bazel build configuration on the `cpp` branch is fundamentally broken. It suffers from missing external dependency definitions (e.g., `@rules_proto`), preventing any C++ targets from being compiled. Attempts to patch the `BUILD` and `WORKSPACE` files were unsuccessful.
*   **Blocker 2: Missing System Dependencies:** A pivot to a manual build was attempted. However, this was blocked by the lack of required build tools (`bison`, `flex`) in the environment.

Due to these blockers, the C++ approach was abandoned.

### 2. Pivot to a Go Implementation

A new strategy was adopted: rewrite the BQL parser from scratch in Go, using `goyacc` for the parser and TinyGo for WASM compilation.

*   **Successes:**
    *   A new Go project was successfully initialized.
    *   A lexer (`lexer.go`) capable of tokenizing BQL queries (including keywords and strings) was written.
    *   A set of Go structs defining the BQL Abstract Syntax Tree (AST) was created (`ast.go`).
    *   A `goyacc` grammar file (`bql.y`) was created to define the parsing logic.
    *   A comprehensive unit test suite (`parser_test.go`) was developed.

### 3. Grammar Ambiguity (Resolved)

The primary point of failure was in the `goyacc` grammar (`bql.y`).

*   **The Problem:** When parsing queries with multiple `SELECT` expressions but no `WHERE` clause (e.g., `SELECT account, balance`), the parser incorrectly assigned the last select expression to the `WHERE` field of the AST.
*   **Root Cause:** The empty production rule for `where_clause_opt` did not explicitly assign `$$ = Expression{}`. In `goyacc`, failing to assign `$$` in an empty production leaves stale values from the parser stack, causing the last `IDENT` token to leak into the `Where` field.
*   **Fix:** Explicitly set `$$ = Expression{}` in the empty `where_clause_opt` rule. Additionally, the `Where` JSON tag was changed from `omitempty` to always serialize, and the lexer was updated to detect and report unclosed string literals.

### 4. WASM Compilation (Complete)

With all tests passing, the parser was compiled to a WASI-compliant WASM module:

```bash
tinygo build -o bql_parser.wasm -target wasi .
```

The resulting `bql_parser.wasm` file (1.1MB) exports `ParseBQLToJSON(query string) string`.

---

## Resolved Issues Summary

| Issue | Root Cause | Fix |
|---|---|---|
| `balance` leaked into `WHERE` AST field | Missing `$$ = Expression{}` in empty `where_clause_opt` production | Explicit assignment in `bql.y` |
| `Where` field omitted when empty | `json:"where,omitempty"` tag on struct field | Changed to `json:"where"` |
| Unclosed strings not detected | Lexer consumed to EOF without error on missing closing quote | Added EOF check in string scanning loop, set `lexer.err` |
| Parse errors not propagated | `Error()` method only printed to stdout | Stored error in `BQLLexer.err`, checked in `Parse()` |

---

## Wassette Deployment Guide

The BQL parser is intended to be deployed as a [Wassette](https://github.com/microsoft/wassette) component — Microsoft's open-source runtime that runs WebAssembly Components as secure, sandboxed MCP tools for AI agents.

### What Wassette Provides

Wassette bridges the WebAssembly Component Model with the Model Context Protocol (MCP). It:
- Loads a `.wasm` component and introspects its WIT interface
- Translates each exported function into an MCP tool
- Runs the component in a deny-by-default sandbox (no filesystem, network, or environment access unless explicitly granted)
- Exposes tools to MCP-compatible AI agents (GitHub Copilot, Cursor, Claude Code, Gemini CLI)

### Deployment Steps

1. **Define a WIT interface** (`go_bql_parser/wit/world.wit`) describing the `parse-bql-to-json` export with `result<string, string>` return type.
2. **Generate Go bindings** using `wit-bindgen-go` to create typed stubs in a `gen/` directory.
3. **Register the function** in an `init()` block, wiring `ParseBQLToJSON` to the generated export using the `cm.Result` type.
4. **Add dependencies**: `go.bytecodealliance.org/cm` for the Component Model result type.
5. **Build for WASI Preview 2**: `tinygo build -target wasip2 --wit-package ./wit --wit-world bql-parser`.
6. **Install Wassette**: `curl -fsSL https://raw.githubusercontent.com/microsoft/wassette/main/install.sh | bash` (or Homebrew/WinGet).
7. **Run locally**: `wassette run ./bql_parser.wasm --allow-stdio`.
8. **Register as MCP server** in your AI agent of choice.
9. **Load the component**: Tell the agent `Please load the BQL parser component from ./go_bql_parser/bql_parser.wasm`.
10. **Optionally publish** to an OCI registry (`wkg oci push ghcr.io/<org>/bql-parser:1.0.0`) for remote loading.

### Security Profile

The BQL parser is a **pure computation component**. It requires:
- No filesystem access
- No network access
- No environment variables

Only `--allow-stdio` is needed, making it one of the safest possible Wassette components.

### MCP Client Configuration

| Client | Configuration |
|---|---|
| VS Code | `code --add-mcp '{"name":"Wassette","command":"wassette","args":["run"]}'` |
| Cursor | Add `{"mcpServers":{"wassette":{"command":"wassette","args":["run"]}}}` to MCP settings |
| Claude Code | `claude mcp add -- wassette wassette run` |
| Gemini CLI | Add server config to `~/.gemini/settings.json` |

### Reference

- [Wassette GitHub](https://github.com/microsoft/wassette)
- [Wassette Concepts](https://microsoft.github.io/wassette/latest/concepts.html)
- [Wassette Go Example](https://github.com/microsoft/wassette/tree/main/examples/gomodule-go)
- [WebAssembly Component Model](https://component-model.bytecodealliance.org/)
- [WIT Specification](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md)
