package jq

import (
	"testing"
)

func testParseKind(t *testing.T, input string, kind ExpressionKind) {
	t.Helper()
	expr, err := Parse(input)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	if expr.Kind != kind {
		t.Errorf("parse %q: expected kind %v, got %v", input, kind, expr.Kind)
	}
}

func testParseEqual(t *testing.T, input, expected string) {
	t.Helper()
	expr, err := Parse(input)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	got := Format(expr)
	if got != expected {
		t.Errorf("parse %q: expected %q, got %q", input, expected, got)
	}
}

func testParseError(t *testing.T, input string) {
	t.Helper()
	_, err := Parse(input)
	if err == nil {
		t.Errorf("parse %q: expected error, got nil", input)
	}
}

func testParseBinaryOp(t *testing.T, input string, op BinaryOp) {
	t.Helper()
	expr, err := Parse(input)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	if expr.Kind != KindBinary {
		t.Fatalf("parse %q: expected KindBinary, got %v", input, expr.Kind)
	}
	if expr.BinaryOp() != op {
		t.Errorf("parse %q: expected op %v, got %v", input, op, expr.BinaryOp())
	}
}

func testParseString(t *testing.T, input, expected string) {
	t.Helper()
	expr, err := Parse(input)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	if expr.Kind != KindString {
		t.Fatalf("parse %q: expected KindString, got %v", input, expr.Kind)
	}
	parts := expr.StringParts()
	var got string
	for _, part := range parts {
		if t, ok := part.(StringText); ok {
			got += t.Text
		}
	}
	if got != expected {
		t.Errorf("parse %q: expected %q, got %q", input, expected, got)
	}
}

func TestParseIdentity(t *testing.T) {
	testParseKind(t, ".", KindIdentity)
	testParseKind(t, " . ", KindIdentity)
	testParseEqual(t, ".", ".")
}

func TestParseRecursiveDescent(t *testing.T) {
	testParseKind(t, "..", KindRecursiveDescent)
	testParseEqual(t, "..", "..")
}

func TestParseField(t *testing.T) {
	testParseKind(t, ".foo", KindField)
	testParseKind(t, ".foo.bar", KindField)
	testParseEqual(t, ".foo", ".foo")
	testParseEqual(t, ".foo.bar", ".foo.bar")

	// Quoted field access
	testParseKind(t, `."foo$"`, KindField)
	testParseEqual(t, `."foo$"`, `."foo$"`)
}

func TestParseFieldOptional(t *testing.T) {
	expr, err := Parse(".foo?")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindOptional {
		t.Fatalf("expected KindOptional, got %v", expr.Kind)
	}
	arg := expr.OptionalArg()
	if arg == nil || arg.Kind != KindField {
		t.Fatalf("expected optional field, got %v", arg)
	}
	if arg.FieldName() != "foo" {
		t.Errorf("expected field 'foo', got %q", arg.FieldName())
	}
}

func TestParseIndex(t *testing.T) {
	testParseKind(t, ".[0]", KindIndex)
	testParseKind(t, ".[-1]", KindIndex)
	testParseKind(t, `.["foo"]`, KindIndex)
	testParseEqual(t, ".[0]", ".[0]")
	testParseEqual(t, `.["foo"]`, `.["foo"]`)
}

func TestParseSlice(t *testing.T) {
	testParseKind(t, ".[2:4]", KindSlice)
	testParseKind(t, ".[:3]", KindSlice)
	testParseKind(t, ".[-2:]", KindSlice)
	testParseEqual(t, ".[2:4]", ".[2:4]")
	testParseEqual(t, ".[:3]", ".[:3]")
	testParseEqual(t, ".[-2:]", ".[-2:]")
}

func TestParseIterator(t *testing.T) {
	testParseKind(t, ".[]", KindIterator)
	testParseEqual(t, ".[]", ".[]")

	expr, err := Parse(".[]?")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindOptional {
		t.Fatalf("expected KindOptional, got %v", expr.Kind)
	}
}

func TestParsePipe(t *testing.T) {
	testParseKind(t, ".foo | .bar", KindPipe)
	testParseEqual(t, ".foo | .bar", ".foo | .bar")
	testParseEqual(t, ".a.b.c", ".a.b.c")
}

func TestParseComma(t *testing.T) {
	testParseKind(t, ".foo, .bar", KindComma)
	testParseEqual(t, ".foo, .bar", ".foo, .bar")
}

func TestParseAlternative(t *testing.T) {
	testParseKind(t, ".foo // .bar", KindAlternative)
	testParseEqual(t, ".foo // .bar", ".foo // .bar")
	testParseEqual(t, ".foo // .bar // .baz", ".foo // .bar // .baz")
}

func TestParseArithmetic(t *testing.T) {
	testParseBinaryOp(t, ".a + 1", OpAdd)
	testParseBinaryOp(t, ".a - 1", OpSub)
	testParseBinaryOp(t, ".a * 2", OpMul)
	testParseBinaryOp(t, ".a / 2", OpDiv)
	testParseBinaryOp(t, ".a % 2", OpMod)
	testParseEqual(t, ".a + 1", ".a + 1")
	testParseEqual(t, "10 / . * 3", "10 / . * 3")
}

func TestParseComparison(t *testing.T) {
	testParseBinaryOp(t, ".a == 1", OpEq)
	testParseBinaryOp(t, ".a != 1", OpNe)
	testParseBinaryOp(t, ".a < 1", OpLt)
	testParseBinaryOp(t, ".a > 1", OpGt)
	testParseBinaryOp(t, ".a <= 1", OpLe)
	testParseBinaryOp(t, ".a >= 1", OpGe)
	testParseEqual(t, ". == 0", ". == 0")
}

func TestParseBoolean(t *testing.T) {
	testParseBinaryOp(t, ".a and .b", OpAnd)
	testParseBinaryOp(t, ".a or .b", OpOr)
	testParseEqual(t, ".a and .b", ".a and .b")
	testParseEqual(t, ".a or .b", ".a or .b")
}

func TestParseLiterals(t *testing.T) {
	testParseKind(t, "42", KindNumber)
	testParseKind(t, "3.14", KindNumber)
	testParseKind(t, "1e10", KindNumber)
	testParseKind(t, "true", KindBool)
	testParseKind(t, "false", KindBool)
	testParseKind(t, "null", KindNull)
	testParseEqual(t, "42", "42")
	testParseEqual(t, "3.14", "3.14")
	testParseEqual(t, "1e10", "1e10")
	testParseEqual(t, "true", "true")
	testParseEqual(t, "false", "false")
	testParseEqual(t, "null", "null")
}

func TestParseNegate(t *testing.T) {
	expr, err := Parse("-5")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindNegate {
		t.Fatalf("expected KindNegate, got %v", expr.Kind)
	}
}

func TestParseString(t *testing.T) {
	testParseString(t, `"hello"`, "hello")
	testParseString(t, `"hello\nworld"`, "hello\nworld")
	testParseString(t, `"tab\there"`, "tab\there")
	testParseString(t, `"quote\"here"`, `quote"here`)
	testParseString(t, `"backslash\\here"`, `backslash\here`)
	testParseEqual(t, `"hello"`, `"hello"`)
}

func TestParseStringUnicode(t *testing.T) {
	testParseString(t, `"\u0041"`, "A")
	testParseString(t, `"\u00e9"`, "é")
	// Surrogate pair: U+1F600 (grinning face)
	testParseString(t, `"\ud83d\ude00"`, "😀")
}

func TestParseStringInterpolation(t *testing.T) {
	expr, err := Parse(`"The input was \(.), which is one less than \(.+1)"`)
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindString {
		t.Fatalf("expected KindString, got %v", expr.Kind)
	}
	parts := expr.StringParts()
	count := 0
	for _, part := range parts {
		if _, ok := part.(StringInterp); ok {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 interpolation parts, got %d", count)
	}
}

func TestParseVariable(t *testing.T) {
	testParseKind(t, "$foo", KindVariable)
	testParseKind(t, "$__loc__", KindVariable)
	testParseEqual(t, "$foo", "$foo")
}

func TestParseArray(t *testing.T) {
	testParseKind(t, "[]", KindArray)
	testParseKind(t, "[1, 2, 3]", KindArray)
	testParseEqual(t, "[]", "[]")
	testParseEqual(t, "[1, 2, 3]", "[1, 2, 3]")
	testParseEqual(t, "[.user, .projects[]]", "[.user, .projects[]]")
}

func TestParseObject(t *testing.T) {
	testParseKind(t, "{}", KindObject)
	testParseKind(t, "{a: 42}", KindObject)
	testParseEqual(t, "{}", "{}")
	testParseEqual(t, "{a: 42}", "{a: 42}")
	testParseEqual(t, "{a: 42, b: 17}", "{a: 42, b: 17}")
}

func TestParseObjectShorthand(t *testing.T) {
	// {foo} shorthand for {foo: .foo}
	expr, err := Parse("{foo}")
	if err != nil {
		t.Fatal(err)
	}
	entries := expr.ObjectEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].KeyKind != ObjectKeyShorthand {
		t.Errorf("expected ObjectKeyShorthand, got %v", entries[0].KeyKind)
	}
	if entries[0].KeyName != "foo" {
		t.Errorf("expected key 'foo', got %q", entries[0].KeyName)
	}
	testParseEqual(t, "{foo}", "{foo}")
}

func TestParseObjectVariableShorthand(t *testing.T) {
	// {$foo} shorthand for {foo: $foo}
	expr, err := Parse("{$foo}")
	if err != nil {
		t.Fatal(err)
	}
	entries := expr.ObjectEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].KeyKind != ObjectKeyShorthand {
		t.Errorf("expected ObjectKeyShorthand, got %v", entries[0].KeyKind)
	}
	if entries[0].KeyName != "foo" {
		t.Errorf("expected key 'foo', got %q", entries[0].KeyName)
	}
	testParseEqual(t, "{$foo}", "{$foo}")
}

func TestParseObjectExpressionKey(t *testing.T) {
	testParseEqual(t, `{("a"+"b"): 59}`, `{("a" + "b"): 59}`)
}

func TestParseObjectVariableKey(t *testing.T) {
	expr, err := Parse("{$bar: $foo}")
	if err != nil {
		t.Fatal(err)
	}
	entries := expr.ObjectEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].KeyKind != ObjectKeyVariable {
		t.Errorf("expected ObjectKeyVariable, got %v", entries[0].KeyKind)
	}
}

func TestParseFunctionCall(t *testing.T) {
	testParseKind(t, "map(. + 1)", KindFunctionCall)
	testParseKind(t, "keys", KindFunctionCall)
	testParseEqual(t, "map(. + 1)", "map(. + 1)")
	testParseEqual(t, "keys", "keys")
	testParseEqual(t, "select(. > 2)", "select(. > 2)")
}

func TestParseIf(t *testing.T) {
	testParseKind(t, "if . == 0 then \"zero\" else \"nonzero\" end", KindIf)
	testParseEqual(t, "if . == 0 then \"zero\" else \"nonzero\" end", "if . == 0 then \"zero\" else \"nonzero\" end")

	// if with elif
	expr, err := Parse("if . == 0 then \"zero\" elif . == 1 then \"one\" else \"many\" end")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindIf {
		t.Fatalf("expected KindIf, got %v", expr.Kind)
	}
	if len(expr.IfElifs()) != 1 {
		t.Errorf("expected 1 elif, got %d", len(expr.IfElifs()))
	}

	// if without else
	testParseEqual(t, "if . > 0 then . end", "if . > 0 then . end")
}

func TestParseTry(t *testing.T) {
	testParseKind(t, "try .a", KindTry)
	testParseKind(t, "try .a catch .", KindTry)
	testParseEqual(t, "try .a", "try .a")
	testParseEqual(t, "try .a catch .", "try .a catch .")
}

func TestParseReduce(t *testing.T) {
	testParseKind(t, "reduce .[] as $item (0; . + $item)", KindReduce)
	testParseEqual(t, "reduce .[] as $item (0; . + $item)", "reduce .[] as $item (0; . + $item)")
}

func TestParseForeach(t *testing.T) {
	testParseKind(t, "foreach .[] as $item (0; . + $item)", KindForeach)
	testParseKind(t, "foreach .[] as $item (0; . + $item; [$item, . * 2])", KindForeach)
	testParseEqual(t, "foreach .[] as $item (0; . + $item)", "foreach .[] as $item (0; . + $item)")
}

func TestParseAsBinding(t *testing.T) {
	testParseKind(t, ".realnames as $names | .posts[]", KindAsBinding)
	testParseEqual(t, ".realnames as $names | .posts[]", ".realnames as $names | .posts[]")
}

func TestParseAsBindingDestructuring(t *testing.T) {
	testParseKind(t, ". as [$a, $b] | $a + $b", KindAsBinding)
	testParseEqual(t, ". as [$a, $b] | $a + $b", ". as [$a, $b] | $a + $b")

	testParseKind(t, ". as {a: $x, b: $y} | $x", KindAsBinding)
}

func TestParseAsBindingDestructuringAlt(t *testing.T) {
	expr, err := Parse(".[] as [$id, $kind] ?// {$id, $kind} | {$id, $kind}")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindAsBinding {
		t.Fatalf("expected KindAsBinding, got %v", expr.Kind)
	}
	pattern := expr.AsPattern()
	if pattern.Kind != KindPatternAlternative {
		t.Fatalf("expected KindPatternAlternative, got %v", pattern.Kind)
	}
	alts := pattern.PatternAlternatives()
	if len(alts) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(alts))
	}
}

func TestParseLabel(t *testing.T) {
	testParseKind(t, "label $out | reduce .[] as $item (0; . + $item)", KindLabel)
	testParseEqual(t, "label $out | .", "label $out | .")
}

func TestParseBreak(t *testing.T) {
	expr, err := Parse("label $out | reduce .[] as $item (0; if . > 10 then break $out else . + $item end)")
	if err != nil {
		t.Fatal(err)
	}
	// Should parse successfully
	if expr.Kind != KindLabel {
		t.Fatalf("expected KindLabel, got %v", expr.Kind)
	}
}

func TestParseDef(t *testing.T) {
	testParseKind(t, "def increment: . + 1; increment", KindDef)
	testParseEqual(t, "def increment: . + 1; increment", "def increment: . + 1; increment")

	// def with args
	testParseKind(t, "def map(f): [.[] | f]; map(. + 1)", KindDef)
	testParseEqual(t, "def map(f): [.[] | f]; map(. + 1)", "def map(f): [.[] | f]; map(. + 1)")

	// def with value args
	testParseKind(t, "def addvalue($f): . + $f; addvalue(1)", KindDef)
}

func TestParseFormat(t *testing.T) {
	testParseKind(t, "@base64", KindFormat)
	testParseKind(t, `@uri "https://example.com"`, KindFormat)
	testParseEqual(t, "@base64", "@base64")
	testParseEqual(t, `@uri "https://example.com"`, `@uri "https://example.com"`)
}

func TestParseFormatWithInterpolation(t *testing.T) {
	testParseEqual(t, `@uri "https://example.com/search?q=\(.search)"`, `@uri "https://example.com/search?q=\(.search)"`)
}

func TestParseComment(t *testing.T) {
	testParseEqual(t, ".foo # comment", ".foo")
	testParseEqual(t, "# whole line comment\n.foo", ".foo")
	testParseEqual(t, ".foo # comment\n| .bar", ".foo | .bar")
}

func TestParsePrecedence(t *testing.T) {
	// Comma has higher precedence than pipe — no parens needed
	testParseEqual(t, "true, false | not", "true, false | not")

	// Alternative has higher precedence than comma — no parens needed
	testParseEqual(t, "false, 1 // 2", "false, 1 // 2")

	// Multiplicative higher than additive — no parens needed
	testParseEqual(t, "1 + 2 * 3", "1 + 2 * 3")

	// Comparison higher than and/or — no parens needed
	testParseEqual(t, ".a > 0 and .b > 0", ".a > 0 and .b > 0")

	// Parens override precedence
	testParseEqual(t, "(1 + 2) * 3", "(1 + 2) * 3")
	testParseEqual(t, ".a | (.b, .c)", ".a | (.b, .c)")
}

func TestParseParenthesized(t *testing.T) {
	testParseEqual(t, "(. + 2) * 5", "(. + 2) * 5")
	testParseEqual(t, "(.foo, .bar) | .baz", "(.foo, .bar) | .baz")
}

func TestParseAssign(t *testing.T) {
	expr, err := Parse(".foo = 42")
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != KindAssign {
		t.Fatalf("expected KindAssign, got %v", expr.Kind)
	}
	if expr.AssignOp() != AssignPlain {
		t.Errorf("expected AssignPlain, got %v", expr.AssignOp())
	}
}

func TestParseUpdateAssign(t *testing.T) {
	ops := []struct {
		input string
		op    AssignOp
	}{
		{".foo |= . + 1", AssignUpdate},
		{".foo += 1", AssignAdd},
		{".foo -= 1", AssignSub},
		{".foo *= 2", AssignMul},
		{".foo /= 2", AssignDiv},
		{".foo %= 2", AssignMod},
		{".foo //= 42", AssignAlt},
	}
	for _, tt := range ops {
		expr, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if expr.Kind != KindUpdateAssign {
			t.Errorf("parse %q: expected KindUpdateAssign, got %v", tt.input, expr.Kind)
		}
		if expr.AssignOp() != tt.op {
			t.Errorf("parse %q: expected op %v, got %v", tt.input, tt.op, expr.AssignOp())
		}
	}
}

func TestParseComplexPrograms(t *testing.T) {
	programs := []string{
		`.realnames as $names | .posts[] | {title, author: $names[.author]}`,
		`[.[] | .name]`,
		`reduce .[] as $item (0; . + $item)`,
		`[.[] | select(.id == "second")]`,
		`{user, title: .titles[]}`,
		`[.. | .a?]`,
		`map(. + 1)`,
		`[.[] | tonumber?]`,
		`if .name == "" then "empty" else .name end`,
		`try .a catch ". is not an object"`,
		`label $out | reduce .[] as $item (null; if . == false then break $out else . + $item end)`,
		`def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while; [while(.<100; .*2)]`,
		`@base64`,
		`@uri "https://www.google.com/search?q=\(.search)"`,
		`. as $big | [$big, $big + 1]`,
		`del(.foo)`,
		`(.a, .b) |= . + 1`,
		`.posts[].comments |= . + ["this is great"]`,
	}
	for _, prog := range programs {
		_, err := Parse(prog)
		if err != nil {
			t.Errorf("parse %q: %v", prog, err)
		}
	}
}

func TestParseErrors(t *testing.T) {
	testParseError(t, "")
	testParseError(t, ".foo |")
	testParseError(t, ".foo .")
	testParseError(t, "if . > 0 then .")
	testParseError(t, "reduce .[] as $item (0)")
	testParseError(t, "[1, 2")
	testParseError(t, "{a: ")
	testParseError(t, "def foo: .")
	testParseError(t, "break")
	testParseError(t, `"unterminated`)
}

func TestParseSpans(t *testing.T) {
	tests := []struct {
		input   string
		spanStr string
	}{
		{" .foo ", ".foo"},
		{" .foo | .bar ", ".foo | .bar"},
		{" ( .foo ) ", "( .foo )"},
		{" [1, 2, 3] ", "[1, 2, 3]"},
	}
	for _, tt := range tests {
		expr, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		got := tt.input[expr.Span.Start:expr.Span.End]
		if got != tt.spanStr {
			t.Errorf("parse %q: span expected %q, got %q", tt.input, tt.spanStr, got)
		}
	}
}
