package cmd

import (
	"testing"

	"github.com/carapace-sh/carapace-jq/pkg/jq"
)

// Realistic jq filter examples sourced from the jq manual
var parseSuccessCases = []string{
	// Basic filters
	".",
	"..",
	".foo",
	".foo.bar",
	".[0]",
	".[-1]",
	".[2:4]",
	".[:3]",
	".[-2:]",
	".[]",
	".foo?",
	".foo[]?",

	// Pipes and commas
	".foo | .bar",
	".[] | .name",
	".foo, .bar",
	"true, false | not",

	// Operators
	".a + 1",
	".a - 1",
	".a * 2",
	".a / 2",
	".a % 2",
	".a // .b",
	".a == 1",
	".a != 1",
	".a < 1",
	".a > 1",
	".a <= 1",
	".a >= 1",
	".a and .b",
	".a or .b",
	"-(.a)",

	// Literals
	"42",
	"3.14",
	"1e10",
	"true",
	"false",
	"null",
	`"hello"`,
	`"hello\nworld"`,
	`"The input was \(.), which is one less than \(.+1)"`,

	// Variables
	"$foo",
	"$__loc__",

	// Arrays and objects
	"[]",
	"[1, 2, 3]",
	"[.user, .projects[]]",
	"{}",
	"{a: 42}",
	"{a: 42, b: 17}",
	"{foo}",
	"{$foo}",
	`{"foo": 42}`,
	`{("a"+"b"): 59}`,
	"{$bar: $foo}",

	// Functions
	"map(. + 1)",
	"select(. > 2)",
	"keys",
	"to_entries",

	// Control flow
	"if . == 0 then \"zero\" else \"nonzero\" end",
	"if . == 0 then \"zero\" elif . == 1 then \"one\" else \"many\" end",
	"if . > 0 then . end",
	"try .a",
	"try .a catch .",
	"reduce .[] as $item (0; . + $item)",
	"foreach .[] as $item (0; . + $item)",
	"foreach .[] as $item (0; . + $item; [$item, . * 2])",

	// As binding
	".realnames as $names | .posts[]",
	". as [$a, $b] | $a + $b",
	". as {a: $x, b: $y} | $x",
	".[] as [$id, $kind] ?// {$id, $kind} | {$id, $kind}",

	// Label/break
	"label $out | .",
	"label $out | reduce .[] as $item (0; if . > 10 then break $out else . + $item end)",

	// Def
	"def increment: . + 1; increment",
	"def map(f): [.[] | f]; map(. + 1)",
	"def addvalue($f): . + $f; addvalue(1)",
	"def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while; [while(.<100; .*2)]",

	// Format strings
	"@base64",
	`@uri "https://example.com"`,
	`@uri "https://example.com/search?q=\(.search)"`,

	// Assignments
	".foo = 42",
	".foo |= . + 1",
	".foo += 1",
	"(.a, .b) |= . + 1",
	".posts[].comments |= . + [\"this is great\"]",

	// Comments
	".foo # comment",
	"# whole line comment\n.foo",

	// Complex programs
	`.realnames as $names | .posts[] | {title, author: $names[.author]}`,
	`[.[] | .name]`,
	`[.. | .a?]`,
	`[.[] | tonumber?]`,
	`if .name == "" then "empty" else .name end`,
	`del(.foo)`,
	// Semicolon as arg separator
	`limit(3; .[])`,
	`range(0; 10; 3)`,
	`INDEX(.[]; .id)`,
	// Keyword field names
	`.if`,
	`."foo bar"`,
	// Pipe right-assoc in sub-expressions
	`[.a | .b | .c]`,
	`{x: .a | .b | .c}`,
	`"\(.a | .b | .c)"`,
	// Non-associative operators (single use, valid)
	`.a == .b`,
	`.a |= .b`,
	`.foo //= .bar`,
}

func TestParseAllSuccessCases(t *testing.T) {
	for _, prog := range parseSuccessCases {
		_, err := jq.Parse(prog)
		if err != nil {
			t.Errorf("parse %q: %v", prog, err)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	for _, prog := range parseSuccessCases {
		expr, err := jq.Parse(prog)
		if err != nil {
			t.Errorf("parse %q: %v", prog, err)
			continue
		}
		formatted := jq.Format(expr)
		_, err = jq.Parse(formatted)
		if err != nil {
			t.Errorf("re-parse %q (from %q): %v", formatted, prog, err)
		}
	}
}

func TestCompletionAllCases(t *testing.T) {
	cases := []string{
		"",
		".",
		".foo",
		".foo | ",
		"map(",
		"[",
		"{",
		"{foo: ",
		"if . > 0 then ",
		"try ",
		"reduce .[] as $item (",
		"def ",
		`"hello \(`,
		"@",
	}
	for _, input := range cases {
		ctx := jq.ParseForCompletion(input)
		if len(ctx.ExpectedTokens) == 0 {
			t.Errorf("completion %q: no expected tokens", input)
		}
	}
}
