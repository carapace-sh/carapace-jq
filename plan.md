# Plan: jq Filter Lexer for carapace-jq

## Goal

Build a Go library that parses jq filter expressions into an AST, with completion support — following the same architecture as `carapace-jjlex` (revset/fileset/template parsers) and `carapace-ffmpeg` (filtergraph/streamspec/argstream parsers). The library enables context-aware completion of jq filter expressions in [carapace-bin].

## Architecture Overview

One parser package (`pkg/jq/`), a CLI (`cmd/carapace-jq/`), and completion actions (`pkg/actions/tools/jq/`). The parser package contains two parsers that share helper functions but have independent types:

- **Full parser** (`parser.go`) — `Parse(input)` → `*Expression` AST with spans. Strict: rejects partial/invalid input.
- **Completion parser** (`completion_parser.go`) — `ParseForCompletion(input)` → `*CompletionContext` describing what tokens are valid at the cursor. Tolerant: recovers from errors to report expectations.

Both parsers implement the same operator precedence and grammar but independently. The completion parser mirrors the main parser's structure but stops at the cursor and records expectations instead of building a full AST.

## Package Structure

```
carapace-jq/
├── cmd/carapace-jq/
│   ├── main.go                    # Entry point, calls cmd.Execute()
│   └── cmd/
│       ├── root.go                # Root cobra command
│       ├── filter.go              # `filter` and `filter-complete` subcommands
│       └── root_test.go           # Integration tests
├── pkg/
│   ├── jq/                        # Core parser package
│   │   ├── span.go                # Span (Start/End byte offsets), Pos
│   │   ├── ast.go                 # AST node types, payload structs, accessors
│   │   ├── scanner.go             # Scanner helpers (identifiers, numbers, strings, comments)
│   │   ├── parser.go              # Full parser: Parse(), IsIdentifier(), Format()
│   │   ├── format.go              # AST → string with precedence-aware parenthesization
│   │   ├── completion.go          # CompletionContext, ExpectedToken, ValidOperator, FunctionContext
│   │   ├── completion_parser.go   # Completion parser: ParseForCompletion()
│   │   ├── jq_test.go             # Parser tests
│   │   ├── completion_test.go     # Completion parser tests
│   │   └── edgecase_test.go       # Edge-case parse tests from jq manual/cookbook
│   └── actions/tools/jq/
│       ├── builtins.go            # Static action definitions (builtin functions, keywords, operators)
│       ├── completion.go          # Completion wiring: maps CompletionContext → carapace actions
│       ├── uid.go                 # UID generation for action deduplication
│       ├── completion_test.go     # Action unit tests (format values, no double-at)
│       └── sandbox_test.go        # Sandbox tests for ActionFilters at cursor positions
├── man/jq/                        # YAML man pages for completion value UIDs
├── skills/jq/                     # (existing) AI agent reference documentation
├── go.mod
├── go.sum
├── .goreleaser.yml
├── .github/
│   ├── FUNDING.yml
│   ├── dependabot.yml
│   └── workflows/
│       ├── dependabot.yml
│       └── go.yml
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── README.md
└── plan.md                        # This file
```

## jq Grammar

### Operator Precedence (lowest to highest)

From jq's `src/parser.y` bison precedence declarations:

| Level | Operators | Associativity | Notes |
|-------|-----------|---------------|-------|
| 1 | `def name: body; rest` | — | Function definition (grammar construct) |
| 2 | `\|` | Right | Pipe |
| 3 | `,` | Left | Comma — concatenate output streams |
| 4 | `//` | Right | Alternative operator |
| 5 | `=` `\|=` `+=` `-=` `*=` `/=` `%=` `//=` | Non-assoc | Assignment / update-assignment |
| 6 | `or` | Left | Boolean OR |
| 7 | `and` | Left | Boolean AND |
| 8 | `==` `!=` `<` `>` `<=` `>=` | Non-assoc | Comparison |
| 9 | `+` `-` | Left | Additive (binary); unary `-` is prefix |
| 10 | `*` `/` `%` | Left | Multiplicative |
| 11 | `?` `.` `[` | — | Postfix: error suppression, field access, indexing |
| 12 | `try` | — | Try expression |
| 13 | `catch` | — | Catch expression (highest) |

Grammar-level constructs (not precedence-based, handled at `Query` level):

| Construct | Syntax |
|-----------|--------|
| Variable binding | `exp as $var \| rest` |
| Variable binding with destructuring | `exp as {a: $x, b: $y} \| rest` |
| Destructuring alternative | `exp as [$a] ?// {$a} \| rest` |
| Label | `label $name \| rest` |
| Conditional | `if cond then a elif cond2 then b else c end` |
| Try/catch | `try exp catch handler` |
| Try (no catch) | `try exp` |
| Reduce | `reduce exp as $var (init; update)` |
| Foreach | `foreach exp as $var (init; update; extract)` |
| Function definition | `def name: body;` / `def name(args): body;` |
| Import | `import "file" as $name;` / `import "file" as name;` |
| Include | `include "file";` |
| Module | `module {metadata};` |

### Token Types

| Category | Tokens |
|----------|--------|
| **Literals** | numbers (`42`, `3.14`, `1e10`), strings (`"..."` with interpolation), `true`, `false`, `null` |
| **Identity** | `.`, `..` (recursive descent) |
| **Field access** | `.foo`, `."foo"`, `.["foo"]` |
| **Indexing** | `.[0]`, `.[-1]`, `.[2:4]`, `.[:3]`, `.[-2:]`, `.[]` |
| **Operators** | `\|`, `,`, `//`, `=`, `\|=`, `+=`, `-=`, `*=`, `/=`, `%=`, `//=`, `==`, `!=`, `<`, `>`, `<=`, `>=`, `+`, `-`, `*`, `/`, `%`, `?` |
| **Keywords** | `if`, `then`, `else`, `elif`, `end`, `try`, `catch`, `reduce`, `foreach`, `as`, `def`, `import`, `include`, `module`, `label`, `break`, `and`, `or`, `not`, `true`, `false`, `null` |
| **Variables** | `$name`, `$__loc__`, `$ENV`, `$ARGS` |
| **Format strings** | `@base64`, `@base64d`, `@csv`, `@tsv`, `@html`, `@uri`, `@sh`, `@text`, `@json` |
| **Punctuation** | `(`, `)`, `[`, `]`, `{`, `}`, `:`, `;`, `$`, `@` |
| **Comments** | `#` to end of line |

### String Interpolation

Strings can contain `\(...)` which embeds a full jq expression:

```jq
"The input was \(.), which is one less than \(.+1)"
@uri "https://example.com/search?q=\(.search)"
```

The expression inside `\(...)` is a full filter — it can contain nested strings with their own interpolation. This requires recursive parsing within string literals.

### Object Construction Shorthands

| Syntax | Meaning |
|--------|---------|
| `{foo: 42}` | Key `foo` with value `42` |
| `{foo}` | Shorthand for `{foo: .foo}` |
| `{$foo}` | Shorthand for `{foo: $foo}` (variable name → key) |
| `{("a"+"b"): 59}` | Expression key (must be parenthesized) |
| `{$bar: $foo}` | Variable value as key: `{($bar): $foo}` |

### AST Node Types

```
ExpressionKind:
  KindIdentity           — .
  KindRecursiveDescent  — ..
  KindField             — .foo, ."foo"
  KindIndex             — .[expr]
  KindSlice             — .[start:end]
  KindIterator          — .[]
  KindOptional          — exp?
  KindPipe              — lhs | rhs
  KindComma             — lhs, rhs
  KindAlternative       — lhs // rhs
  KindAssign            — lhs = rhs
  KindUpdateAssign      — lhs |= rhs  (and +=, -=, *=, /= %=, //=)
  KindBinary            — lhs op rhs  (+, -, *, /, %, ==, !=, <, >, <=, >=, and, or)
  KindNegate            — -exp  (unary minus)
  KindNumber            — 42, 3.14, 1e10
  KindString            — "..."  (with interpolation parts)
  KindBool              — true, false
  KindNull              — null
  KindVariable          — $name
  KindArray             — [exp]
  KindObject            — {key: value, ...}
  KindFunctionCall      — name(args)
  KindFormat            — @base64, @uri "..."
  KindIf                — if cond then ... elif ... else ... end
  KindTry               — try exp catch handler
  KindReduce            — reduce exp as $var (init; update)
  KindForeach           — foreach exp as $var (init; update; extract)
  KindAsBinding         — exp as $var | rest
  KindLabel             — label $name | rest
  KindBreak             — break $name
  KindDef               — def name(args): body; rest
  KindParenthesized     — (exp)
```

## Implementation Phases

### Phase 1: Scanner + AST + Span + Format

**Files:** `span.go`, `ast.go`, `scanner.go`, `format.go`

Scanner helpers on the parser struct (embedded approach, matching jjlex pattern):
- `skipWhitespace()` — skip spaces, tabs, newlines (NOT comments — comments handled separately)
- `skipComment()` — skip `#` to end of line
- `skipWhitespaceAndComments()` — combined
- `scanIdentifier()` — `[a-zA-Z_][a-zA-Z0-9_]*`
- `scanNumber()` — integers, decimals, exponents
- `parseStringLiteral()` — double-quoted strings with escape sequences and `\(...)` interpolation
- `scanFormatName()` — `@[a-zA-Z0-9_]+`

AST: type-erased payload pattern (same as revset/template) — `Expression.payload` is `any`, accessors check `Kind`.

Format: precedence-aware parenthesization for round-tripping AST → string.

### Phase 2: Full Parser

**Files:** `parser.go`, `jq_test.go`

Recursive descent parser following jq's precedence hierarchy. Pratt parser for the binary operator levels (matching the template parser approach), with grammar-level constructs handled at the top level.

```
Parse(input) → *Expression

parseQuery()           — top level: def | import | include | module | as-binding | label | pipe-level
parsePipe()            — | (right-assoc)
parseComma()           — , (left-assoc)
parseAlternative()     — // (right-assoc)
parseAssignment()      — =, |=, +=, -=, *=, /=, %=, //= (non-assoc)
parseOr()              — or (left-assoc)
parseAnd()             — and (left-assoc)
parseComparison()      — ==, !=, <, >, <=, >= (non-assoc)
parseAdditive()        — +, - (left-assoc)
parseMultiplicative()  — *, /, % (left-assoc)
parsePostfix()         — ?, .foo, .["foo"], .[0], .[2:4], .[] (chained)
parsePrefix()          — - (unary minus)
parsePrimary()         — literals, keywords, function calls, parens, arrays, objects
```

Control flow constructs parsed at the primary level:
- `parseIf()` — if/then/else/elif/end
- `parseTry()` — try/catch
- `parseReduce()` — reduce ... as ... (init; update)
- `parseForeach()` — foreach ... as ... (init; update; extract)
- `parseAsBinding()` — exp as $pattern | rest (at query level)
- `parseLabel()` — label $name | rest (at query level)
- `parseDef()` — def name(args): body; rest (at query level)

Object construction:
- `parseObject()` — `{key: value, ...}` with shorthand handling
  - Bare key: `{foo: ...}` → key is identifier
  - Variable shorthand: `{$foo}` → key is "foo", value is $foo
  - Value shorthand: `{foo}` → key is "foo", value is .foo
  - Expression key: `{(expr): ...}` → key from parenthesized expression

String interpolation:
- `parseString()` — parses `"..."` with `\(...)` interpolation parts
  - Text parts: regular characters and escape sequences
  - Interpolation parts: `\(...)` containing a full `parseQuery()` expression
  - Format string prefix: `@foo "..."` — format applies to interpolations

### Phase 3: Completion Parser

**Files:** `completion.go`, `completion_parser.go`, `completion_test.go`

`ParseForCompletion(input)` → `*CompletionContext`

CompletionContext fields (extending the jjlex pattern):

```go
type CompletionContext struct {
    ExpectedTokens []ExpectedToken   // what token types are valid
    ValidOperators []ValidOperator   // which operators are valid
    PartialIdent   string            // partial identifier being typed
    PartialString  string            // partial string content (no quotes)
    StringQuote    rune              // quote char if inside unclosed string
    Function       *FunctionContext  // if inside a function call
    InFormat       bool              // if completing @format name
    InObjectKey    bool              // if completing object key
    InObjectValue  bool              // if completing object value after colon
    InSlice        bool              // if inside .[a:b] slice
    InReduce       *ReduceContext    // if inside reduce/foreach parens
    InAsPattern    bool              // if completing destructuring pattern
    PartialFormat  string            // partial @format name
}
```

ExpectedToken enum (jq-specific):
- `ExpectedExpression` — any primary expression valid
- `ExpectedOperator` — infix/postfix operator valid
- `ExpectedPipe` — `|` expected (after `as $var`, after `label $name`)
- `ExpectedColon` — `:` expected (object key-value, slice)
- `ExpectedSemicolon` — `;` expected (reduce/foreach args, def terminator)
- `ExpectedClosingParen` — `)` expected
- `ExpectedClosingBracket` — `]` expected
- `ExpectedClosingBrace` — `}` expected
- `ExpectedComma` — `,` expected (function args, array, object)
- `ExpectedStringClose` — closing `"` expected
- `ExpectedKeyword` — keyword expected (then, else, elif, end, catch, as)
- `ExpectedFormatName` — `@format` name expected
- `ExpectedDollar` — `$` expected (variable start)

### Phase 4: CLI

**Files:** `cmd/carapace-jq/main.go`, `cmd/carapace-jq/cmd/root.go`, `cmd/carapace-jq/cmd/filter.go`

Subcommands (matching jjlex pattern):
- `carapace-jq filter <expression>` — parse filter, output AST as JSON
- `carapace-jq filter-complete <expression>` — completion context as JSON

### Phase 5: Completion Actions

**Files:** `pkg/actions/tools/jq/builtins.go`, `pkg/actions/tools/jq/completion.go`, `pkg/actions/tools/jq/uid.go`

Map `CompletionContext` to carapace actions:
- `ExpectedExpression` → ActionBuiltins (function names), ActionKeywords (if, try, reduce, etc.), ActionValues (true, false, null), ActionValues (".", "..")
- `ExpectedOperator` → ActionOperators (context-dependent valid operators)
- `Function != nil` → function-arg-specific actions (e.g., `map` arg 0 → any expression, `select` arg 0 → boolean expression)
- `InFormat` → ActionFormatStrings (@base64, @csv, etc.)
- `InObjectKey` → ActionFieldNames (from input JSON, if available)
- `PartialIdent` → filter builtins/keywords by prefix
- `PartialString` → context-dependent string completions

### Phase 6: Man Pages + Metadata

**Files:** `man/jq/*.yaml`, `.goreleaser.yml`, `.github/`, `AGENTS.md`, `README.md`, `CONTRIBUTING.md`

YAML documentation for completion value UIDs (builtin functions, operators, keywords, format strings).

## Key Technical Decisions

### 1. Embedded scanner vs. token stream

**Decision: Embedded scanner (matching jjlex pattern).**

The parser works directly on the input string using `peek()`, `advance()`, `skipWhitespace()`, etc. No separate token stream. This handles string interpolation naturally — `parseString()` can recursively call `parseQuery()` for `\(...)` parts without needing a token buffer.

jq's real implementation uses flex+bison (separate lexer), but the embedded approach is simpler in Go and consistent with the sibling projects.

### 2. Pratt parser vs. layered recursive descent

**Decision: Pratt parser for binary operators, layered for grammar constructs.**

Following the template parser pattern: `parsePratt(minPrec)` handles all binary operators (`|`, `,`, `//`, assignments, `or`, `and`, comparisons, `+`/`-`, `*`/`/`/`%`) with a precedence table. Grammar-level constructs (`if`, `try`, `reduce`, `def`, `as`) are handled at the appropriate level above or within `parsePrimary()`.

The pipe `|` and comma `,` are at the top of the Pratt parser (lowest precedence). However, `as` bindings, `label`, `def`, `import`, `include`, and `module` are grammar-level constructs that interact with `|` in special ways — these are handled at the `parseQuery()` level, above the Pratt parser.

### 3. Scope of Phase 1

**Decision: Parse the full filter expression language including control flow, but defer the module system.**

Phase 1-2 cover: identity, field access, indexing, slicing, iteration, all operators, literals, string interpolation, format strings, function calls, array/object construction, `if`/`try`/`reduce`/`foreach`, `as` binding with destructuring, `label`/`break`, `def`.

Deferred to a later phase: `import`, `include`, `module`, `modulemeta`. These are program-level constructs, not filter expressions — they're less important for completion since most users write single-filter programs on the command line.

### 4. String interpolation representation in AST

**Decision: String AST node holds a list of parts (text + interpolation expressions).**

```go
type StringExpr struct {
    Parts []StringPart
}

type StringPart interface{ stringPart() }

type StringText struct { Text string }       // literal text segment
type StringInterp struct { Expr *Expression } // \(expr) interpolation

func (StringText) stringPart()  {}
func (StringInterp) stringPart() {}
```

Format strings (`@foo "..."`) are a separate AST node that wraps a string:

```go
type FormatExpr struct {
    Name   string      // "base64", "uri", etc.
    String *Expression // the string literal (KindString), or nil for bare @foo
}
```

### 5. Object construction representation

```go
type ObjectEntry struct {
    KeyKind   ObjectKeyKind  // bare, variable-shorthand, value-shorthand, expression
    Key       *Expression    // for expression keys (parenthesized)
    KeyName   string         // for bare/shorthand keys
    Value     *Expression    // nil for shorthands (expanded at eval time)
}

type ObjectExpr struct {
    Entries []ObjectEntry
}
```

### 6. Destructuring patterns

`as` bindings support destructuring patterns: `[$a, $b]`, `{$x, $y}`, and `?//` alternatives. Patterns are a subset of the expression grammar — array patterns look like array construction, object patterns look like object construction but with `$var` entries.

**Decision: Model patterns as a separate AST subset, parsed by `parsePattern()` in the `as` context.** Patterns reuse `KindArray`/`KindObject` nodes but with variable entries (`$name`). The `?//` operator creates a `KindPatternAlternative` node.

### 7. Comment handling

jq supports `#` comments to end of line. The scanner skips these like whitespace. This is a difference from the jjlex parsers which don't have comments.

### 8. UTF-8 handling

**Decision: Use `utf8.DecodeRuneInString` for all character access (matching jjlex pattern).** This ensures correct handling of non-ASCII identifiers and string content. jq identifiers are ASCII-only (`[a-zA-Z_][a-zA-Z0-9_]*`), but string literals can contain any Unicode.

## Testing Strategy

Following the jjlex pattern:

- **`jq_test.go`** — Parser tests using helper functions (`testParseKind`, `testParseEqual`, `testParseError`, `testParseString`, span tests). Test round-tripping: `Parse(input).String() == canonical_form`.
- **`completion_test.go`** — Completion tests using `assertHasExpected`, `assertHasOperator`, `assertNoOperator`. Test partial expressions, function args, string interpolation, object construction.
- **`root_test.go`** — Integration tests with realistic jq programs sourced from the jq manual and real-world usage.
- **`actions/tools/jq/action_test.go`** — Sandbox tests for carapace actions (builtin function completion, operator completion).

All tests use only the standard `testing` package (no testify or other deps), matching the sibling projects.

## Completion Scenarios

| Input | ExpectedTokens | Context |
|-------|---------------|---------|
| `""` (empty) | Expression | Completing at start of filter |
| `.` | Operator | After identity |
| `.foo` | Operator | After field access (PartialIdent="foo") |
| `.foo \|` | Expression | After pipe |
| `map(` | Expression, ClosingParen | Inside function call |
| `map(.` | Expression, ClosingParen | Inside function arg |
| `select(. >` | Expression | After comparison operator |
| `[` | Expression, ClosingBracket | Inside array construction |
| `{` | ObjectKey, ClosingBrace | Inside object construction |
| `{foo:` | Expression | After object key colon |
| `if . > 0 then` | Expression | After `then` keyword |
| `if . > 0 then . else` | Expression | After `else` keyword |
| `reduce .[] as $item (0;` | Expression | After `;` in reduce |
| `try .` | Operator, Catch | After try expression |
| `@` | FormatName | After `@` (format string prefix) |
| `"hello \` | Expression | Inside string interpolation |
| `.foo \| ma` | Expression | Partial function name (PartialIdent="ma") |
| `def my` | DefName, Colon | After `def` keyword |

## Open Questions

1. **Module system scope**: Should `import`/`include`/`module` be in Phase 1 or deferred? These are rare in command-line usage but needed for `.jq` library files. **Proposed: defer to a follow-up phase.**

2. **Dynamic field completion**: Should the completion actions try to probe the input JSON to suggest field names for `.foo` completion? This requires access to the input file/stream. **Proposed: yes, but as an optional feature — the completion context provides the information, the action layer decides whether to probe.**

3. **`$__loc__` handling**: The `$__loc__` variable returns a `{file, line}` object. Should it be a special AST node or just a variable? **Proposed: just a variable — it's a builtin binding, not syntactically special.**

4. **`@foo` without string**: `@base64` alone (without a string) applies the format to `.`. Should this be a separate node or `FormatExpr{String: nil}`? **Proposed: `FormatExpr{String: nil}` — simpler, matches the grammar.**

5. **Comment preservation**: Should the AST preserve comments for formatting? **Proposed: no — comments are skipped by the scanner and not preserved. Matches jq's own behavior and the jjlex pattern.**

6. **Number literal preservation**: jq preserves the original literal form of numbers (e.g., `1.000` stays `1.000`). Should the AST store the original text or a parsed number? **Proposed: store the original text string — matches jq's literal preservation semantics.**

7. **Package name**: `pkg/jq/` vs `pkg/filter/`? **Proposed: `pkg/jq/` — the project is `carapace-jq`, and the parser handles the full jq filter language, not just "filters" in a narrow sense.**

## Known Parser Limitations

Identified by sandbox testing against complex jq expressions sourced from the [jq manual](https://jqlang.github.io/jq/manual/) and [jq Cookbook](https://github.com/jqlang/jq/wiki/jq-Cookbook). Each limitation has corresponding skipped test cases in `pkg/jq/sandbox_test.go`.

### 1. Comma inside bracket access

**Affected expressions:** `del(.[1, 2])`, `path(.[1, 2])`

**Root cause:** `parseBracketAccess` (`parser.go:869`) calls `parseExp()` for the index expression, which parses pipe-level expressions but not commas. In jq, `.[1, 2]` is valid — the comma operator generates multiple index values within the bracket.

**Fix:** Use `parseQuery()` or a comma-aware variant instead of `parseExp()` for bracket index expressions. Must also update the completion parser's bracket access handling to match.

**Priority:** Medium — used in real-world `del` and `path` expressions, but uncommon in interactive completion scenarios.

### 2. Module system (`import` / `include` / `module`)

**Affected expressions:** `include "library"`, `import "library" as lib`, `import "library" as lib; lib::walk(.)`

**Root cause:** `parseQuery` (`parser.go:168`) explicitly rejects these with `"module system not yet supported"`. This was a deliberate deferral (see Open Question 1 and Phase 1 scope decision).

**Fix:** Add `parseImport()`, `parseInclude()`, and `parseModule()` to `parseQuery()`. Handle `include "file" {search: "..."}` metadata blocks, `import "file" as $name` / `import "file" as Name` (module vs variable), and `Name::function` scope resolution in postfix/primary parsing.

**Priority:** Low — rare in command-line usage, primarily relevant for `.jq` library files.

### 3. `try`/`catch` with `if` handler

**Affected expression:** `try repeat(exp) catch if .=="break" then empty else error end`

**Root cause:** `parseTry` (`parser.go:1494`) parses the catch handler with `parsePratt(precPostfix)`, which handles binary operators and postfix but not grammar-level constructs like `if`. The `if` expression is parsed at the primary level (`parsePrimary`), which is below the Pratt parser entry point.

**Fix:** Use `parseExp()` or `parseQuery()` for the catch handler instead of `parsePratt(precPostfix)`, allowing full expressions including `if`/`then`/`else`/`end`, `reduce`, `foreach`, `try`, and `def`. Must also update the completion parser's catch handling to match.

**Priority:** Medium — the `try ... catch if ...` pattern appears in real-world jq programs for selective error handling.
