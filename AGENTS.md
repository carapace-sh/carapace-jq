# AGENTS.md

## Project Overview

Go library for parsing [jq](https://jqlang.github.io/jq/) filter expressions into an AST, with completion support. Part of the [carapace-sh](https://github.com/carapace-sh) ecosystem (shell completion framework). The module path is `github.com/carapace-sh/carapace-jq`.

## Commands

```sh
go test ./...                              # run all tests
go test ./pkg/jq/                          # run jq parser package tests only
go test -run TestParse ./pkg/jq/           # run specific test
go build ./...                             # build all packages
go run . filter "<expr>"                   # parse jq filter expression, output AST as JSON
go run . filter-complete "<expr>"          # filter completion context as JSON
```

No Makefile, no linter config, no CI config present.

## Architecture

Cobra-based CLI (`cmd/`) wrapping a recursive-descent parser (`pkg/jq/`) with completion actions that wire the parser to carapace (`pkg/actions/tools/jq/`).

### CLI (`cmd/`)

- **`root.go`** — Root cobra command with `carapace.Gen(rootCmd).Standalone()`
- **`filter.go`** — `filter` and `filter-complete` subcommands

Entry point is `main.go` at `cmd/carapace-jq/`, which calls `cmd.Execute()`.

### Parser (`pkg/jq/`)

- **`parser.go`** — Full parser. `Parse()` → `*Expression` AST with spans. Strict: rejects partial/invalid input.
- **`completion_parser.go`** — Completion parser. `ParseForCompletion(input)` → `*CompletionContext` describing what tokens are valid at end of input. Tolerant: recovers from errors to report expectations.

Both parsers implement the same operator precedence hierarchy but independently. The completion parser mirrors the main parser's structure but stops at end of input and records expectations instead of building a full AST.

### jq operator precedence (lowest to highest)

| Level | Operators | Associativity |
|-------|-----------|---------------|
| 1 | `def name: body; rest` | — |
| 2 | `\|` | Right |
| 3 | `,` | Left |
| 4 | `//` | Right |
| 5 | `=`, `\|=`, `+=`, `-=`, `*=`, `/=`, `%=`, `//=` | Non-assoc |
| 6 | `or` | Left |
| 7 | `and` | Left |
| 8 | `==`, `!=`, `<`, `>`, `<=`, `>=` | Non-assoc |
| 9 | `+`, `-` | Left |
| 10 | `*`, `/`, `%` | Left |
| 11 | `?`, `.`, `[` (postfix) | — |
| 12 | `try` | — |
| 13 | `catch` | — |

### File responsibilities

| File | Purpose |
|---|---|
| `pkg/jq/span.go` | `Span` (Start/End byte offsets) and `Pos` types |
| `pkg/jq/ast.go` | jq AST node types, payload structs, accessor methods |
| `pkg/jq/scanner.go` | Scanner methods (identifiers, numbers, strings, comments) |
| `pkg/jq/parser.go` | jq main parser + public API: `Parse()`, `IsIdentifier()`, `Format()` |
| `pkg/jq/format.go` | jq AST → string formatting with precedence-aware parenthesization |
| `pkg/jq/completion.go` | jq completion context types: `CompletionContext`, `ExpectedToken`, `ValidOperator`, `FunctionContext` |
| `pkg/jq/completion_parser.go` | jq completion parser |
| `pkg/jq/jq_test.go` | jq parser tests |
| `pkg/jq/completion_test.go` | jq completion tests |
| `cmd/carapace-jq/main.go` | CLI entrypoint |
| `cmd/carapace-jq/cmd/root.go` | Root cobra command |
| `cmd/carapace-jq/cmd/filter.go` | Filter subcommands |

## Key Patterns & Gotchas

### Expression uses a type-erased payload pattern

`Expression.payload` is `any`; accessors (`.Identifier()`, `.BinaryLHS()`, etc.) do type checks and return zero values on kind mismatch. Always check `Kind` before calling accessors.

### Two parsers must stay in sync

When modifying operator precedence or parsing rules in `parser.go`, the same changes must be mirrored in `completion_parser.go`. They share helper functions but have independent parser types.

### String interpolation

Strings can contain `\(...)` which embeds a full jq expression. The scanner handles this by recursively parsing the expression inside the interpolation. The AST `StringExpr` holds a list of parts (text segments and interpolation expressions).

### Comments

jq supports `#` comments to end of line. The scanner skips these like whitespace.

### Number literal preservation

jq preserves the original literal form of numbers (e.g., `1.000` stays `1.000`). The AST stores the original text string, not a parsed number.

## Testing

- Tests use standard `testing` package only (no testify or other deps)
- `jq_test.go` — parser tests using helpers
- `completion_test.go` — completion tests using `assertHasExpected`, `assertHasOperator`
- Parser/completion packages have no external dependencies (pure stdlib); `pkg/actions/tools/jq` depends on carapace and cobra

## Skills

The `skills/` directory contains reference documentation for jq concepts and the filter language. The `jq` skill is the user-invocable entrypoint.
