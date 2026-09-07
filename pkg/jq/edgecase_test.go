package jq

// Edge-case parse tests for complex jq expressions sourced from the
// jq manual (https://jqlang.github.io/jq/manual/) and the jq Cookbook
// (https://github.com/jqlang/jq/wiki/jq-Cookbook).
//
// These tests exercise the parser against realistic, complex expressions to
// catch regressions and identify parsing gaps.  Cases that the parser does
// not yet support are marked with t.Skip so they can be enabled once support
// is added.

import (
	"testing"
)

// edgeCase groups a descriptive name with a jq expression and an optional
// skip reason.  When skipReason is non-empty the test is skipped instead of
// failing.
type edgeCase struct {
	name       string
	expr       string
	skipReason string
}

func runEdgeCases(t *testing.T, cases []edgeCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.skipReason != "" {
				t.Skip(c.skipReason)
			}
			_, err := Parse(c.expr)
			if err != nil {
				t.Fatalf("parse %q: %v", c.expr, err)
			}
		})
	}
}

// ----- String interpolation -----

var stringInterpolationCases = []edgeCase{
	{"adjacent interpolations", `"\(.a)\(.b)"`, ""},
	{"interp at start", `"\(.x)hello"`, ""},
	{"interp at end", `"hello\(.x)"`, ""},
	{"only interpolation", `"\(.)"`, ""},
	{"interp with pipe", `"\(.a | .b)"`, ""},
	{"interp with arithmetic", `"\(.+1)"`, ""},
	{"nested string in interp", `"outer \(("inner")) end"`, ""},
	{"multiple interps with text", `"\(.a) and \(.b | .c) plus \(.d)"`, ""},
	{"interp with function call", `"\(map(.x))"`, ""},
	{"interp with array", `"\([.a, .b])"`, ""},
	{"interp with object", `"\({x: .a})"`, ""},
	{"interp with if", `"\(if .a then .b else .c end)"`, ""},
	{"interp with try", `"\(try .a)"`, ""},
	{"interp with reduce", `"\(reduce .[] as $x (0; . + $x))"`, ""},
	{"manual two interps", `"The input was \(.), which is one less than \(.+1)"`, ""},
	{"error with interp", `try error("invalid value: \(.)") catch .`, ""},
	{"loc in string", `"\($__loc__)"`, ""},
	{"loc in error", `try error("\($__loc__)") catch .`, ""},
}

func TestEdgeCaseStringInterpolation(t *testing.T) {
	runEdgeCases(t, stringInterpolationCases)
}

// ----- @format strings -----

var formatCases = []edgeCase{
	{"@base64", "@base64", ""},
	{"@base64d", "@base64d", ""},
	{"@csv", "@csv", ""},
	{"@tsv", "@tsv", ""},
	{"@sh", "@sh", ""},
	{"@html", "@html", ""},
	{"@json", "@json", ""},
	{"@uri", "@uri", ""},
	{"@text", "@text", ""},
	{"@sh with string", `@sh "echo \(.)"`, ""},
	{"@uri with interp", `@uri "https://www.google.com/search?q=\(.search)"`, ""},
	{"@json with interp", `@json "total: \($dot).\n"`, ""},
	{"@sh with multiple interps", `@sh "a=\(.a) b=\(.b)"`, ""},
	{"@base64 then pipe", "@base64 | .", ""},
}

func TestEdgeCaseFormats(t *testing.T) {
	runEdgeCases(t, formatCases)
}

// ----- Numbers -----

var numberCases = []edgeCase{
	{"integer", "42", ""},
	{"decimal", "3.14", ""},
	{"exponent lowercase", "1e10", ""},
	{"exponent uppercase", "1E10", ""},
	{"negative exponent", "1e-10", ""},
	{"positive exponent", "1e+10", ""},
	{"decimal with exponent", "100e-2", ""},
	{"trailing zeros preserved", "1.000", ""},
	{"zero", "0", ""},
	{"many decimal places", "0.12345678901234567890123456789", ""},
	{"very large exponent", "1e308", ""},
	{"negative number", "-5", ""},
	{"double negative", "--5", ""},
	{"negate in expression", "1 + -2", ""},
	{"negate parenthesized", "-(.a + .b)", ""},
	{"negate after pipe", ".a | -.b", ""},
}

func TestEdgeCaseNumbers(t *testing.T) {
	runEdgeCases(t, numberCases)
}

// ----- Chained postfix operations -----

var postfixCases = []edgeCase{
	{"field then iterator then field", ".foo[].bar", ""},
	{"index then field", ".[0].bar", ""},
	{"iterator then optional", ".[]?", ""},
	{"field then slice", ".foo[1:3]", ""},
	{"postfix after paren with pipe", "(.a | .b)[0]", ""},
	{"postfix after paren with comma", "(.a, .b).c", ""},
	{"iterator after paren", "(.a, .b)[]", ""},
	{"optional after paren", "(.a, .b)?", ""},
	{"postfix after function call", "keys[0]", ""},
	{"postfix after array literal", "[1, 2, 3][0]", ""},
	{"index on string literal", `"hello"[0]`, ""},
	{"slice on parenthesized", "(.a)[1:3]", ""},
	{"field then field then field", ".a.b.c", ""},
	{"chained quoted field", `.foo."bar".baz`, ""},
	{"recursive descent then field", ".. | .foo", ""},
	{"recursive descent then optional", ".. | .a?", ""},
	{"recursive in array", "[.. | .]", ""},
	{"recursive with select", `.. | select(type=="object")`, ""},
	{"recursive with objects and keys", `[.. | objects | keys[]] | unique`, ""},
}

func TestEdgeCasePostfix(t *testing.T) {
	runEdgeCases(t, postfixCases)
}

// ----- Variable then postfix -----

var variablePostfixCases = []edgeCase{
	{"var then field", "$obj.foo", ""},
	{"var then index", "$arr[0]", ""},
	{"var then iterator", "$arr[]", ""},
	{"var then slice", "$arr[1:3]", ""},
	{"var then optional", "$arr?", ""},
	{"var then field then field", "$obj.foo.bar", ""},
	{"$ENV then field", "$ENV.HOME", ""},
	{"var in array", "[$foo, $bar]", ""},
	{"var in object", "{$foo, bar: $baz}", ""},
	{"var in arithmetic", "$x + $y", ""},
	{"var in pipe", "$x | .a", ""},
	{"var in comparison", "$x == $y", ""},
	{"var as function arg", "map($x)", ""},
}

func TestEdgeCaseVariablePostfix(t *testing.T) {
	runEdgeCases(t, variablePostfixCases)
}

// ----- Destructuring patterns -----

var destructuringCases = []edgeCase{
	{"array destructure", ". as [$a, $b] | $a + $b", ""},
	{"object destructure", ". as {a: $x, b: $y} | $x", ""},
	{"nested obj in array", ". as [$a, {b: $b}] | $a + $b", ""},
	{"nested array in obj", ". as {a: [$x, $y]} | $x + $y", ""},
	{"deep nested destructure", `. as {$a, $b, c: {$d, $e}} ?// {$a, $b, c: [{$d, $e}]} | {$a, $b, $d, $e}`, ""},
	{"complex event destructure", `.resources[] as {$id, $kind, events: {$user_id, $ts}} ?// {$id, $kind, events: [{$user_id, $ts}]} | {$user_id, $kind, $id, $ts}`, ""},
	{"array then object alternative", ".[] as [$id, $kind] ?// {$id, $kind} | {$id, $kind}", ""},
	{"nested with optional fallback", `[[3]] | .[] as [$a] ?// [$b] | if $a != null then error("err: \($a)") else {$a,$b} end`, ""},
	{"destructuring with pipe in source", `reduce .[] as [$i,$j] (0; . + $i * $j)`, ""},
	{"object destructure in reduce", `reduce .[] as {$x,$y} (null; .x += $x | .y += [$y])`, ""},
	{"complex destructure with nested array/obj", `.[] as {$a, $b, c: {$d}} ?// {$a, $b, c: [{$e}]} | {$a, $b, $d, $e}`, ""},
}

func TestEdgeCaseDestructuring(t *testing.T) {
	runEdgeCases(t, destructuringCases)
}

// ----- Def definitions -----

var defCases = []edgeCase{
	{"simple def", "def increment: . + 1; increment", ""},
	{"def with filter arg", "def map(f): [.[] | f]; map(. + 1)", ""},
	{"def with value arg", "def addvalue($f): . + $f; addvalue(1)", ""},
	{"def with two filter args", "def addvalue(f): f as $x | map(. + $x); addvalue(.[0])", ""},
	{"def with mixed args", "def foo(f; $v): f | . + $v; foo(.a; 5)", ""},
	{"def with three semicolon args", "def foo(a;b;c): a | b | c; foo(.x; .y; .z)", ""},
	{"def with nested def", "def f: def g: . + 1; g; f", ""},
	{"two nested defs in def body", "def f: def g: .; def h: . + 1; g | h; f", ""},
	{"two defs at top level", "def a: . + 1; def b: . + 2; a | b", ""},
	{"nested def with rest after", "def f: def g: .; def h: . + 2; g | h; f", ""},
	{"def with nested def and call after", "def outer: def inner: . + 1; inner; outer", ""},
	{"def with if in nested def", `def f: def g: if . then 1 else 2 end; g; f`, ""},
	{"def with sub in if in nested def", `def f: def g: if type == "string" then sub("^ +";"") else . end; g; f`, ""},
	{"def with two nested defs and sub", `def objectify(headers): def tonumberq: tonumber? // .; def trimq: if type == "string" then sub("^ +";"") | sub(" +$";"") else . end; trimq; objectify([])`, ""},
	{"while definition", "def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while; [while(.<100; .*2)]", ""},
	{"def then label", "def f: .; label $out | f", ""},
	{"def with args and nested def", "def f(x): def g: . + 1; g | x; f(.)", ""},
	{"recursive def body", `def recurse(f): def r: ., (f | select(. != null) | r); r; recurse(.children[])`, ""},
	{"range with nested def", `def range(init; upto; by): def _range: if (by > 0 and . < upto) or (by < 0 and . > upto) then ., ((.+by)|_range) else empty end; if init == upto then empty elif by == 0 then init else init|_range end; range(0; 10; 3)`, ""},
}

func TestEdgeCaseDef(t *testing.T) {
	runEdgeCases(t, defCases)
}

// ----- Reduce / foreach -----

var reduceForeachCases = []edgeCase{
	{"basic reduce", "reduce .[] as $item (0; . + $item)", ""},
	{"reduce with destructure", "reduce .[] as [$i,$j] (0; . + $i * $j)", ""},
	{"reduce with object destructure", `reduce .[] as {$x,$y} (null; .x += $x | .y += [$y])`, ""},
	{"reduce with pipe in source", "reduce (.a | .b) as $x (0; . + $x)", ""},
	{"reduce with complex init", "reduce .[] as $item ({}; .[$item|type] += 1)", ""},
	{"reduce with object init", "reduce .[] as $x ({}; .[$x] += 1)", ""},
	{"reduce with array init", "reduce .[] as $x ([]; . + [$x])", ""},
	{"reduce with pipe in update", "reduce .[] as $x (0; . + $x | . * 2)", ""},
	{"reduce with empty init", "reduce .[] as $x (empty; . + $x)", ""},
	{"reduce with paths", `reduce paths as $p (.; getpath($p) as $v | if $v|type == "string" and $dict[0][$v] then setpath($p; $dict[0][$v]) else . end)`, ""},
	{"basic foreach", "foreach .[] as $item (0; . + $item)", ""},
	{"foreach with extract", "foreach .[] as $item (0; . + $item; [$item, . * 2])", ""},
	{"foreach with object extract", `foreach .[] as $x (0; . + 1; {index: ., $item})`, ""},
	{"foreach with pipe in extract", "foreach .[] as $x (0; . + $x; . | [$x, .])", ""},
	{"foreach with empty update", "foreach .[] as $x (0; .; .)", ""},
	{"foreach with complex source", "foreach (inputs, null) as $line (0; .; .)", ""},
	{"foreach with conditional in update", `foreach s as $x (null; if . == null or .emitted != $x then {emit: true, emitted: $x} else .emit = false end; if .emit then $x else empty end)`, ""},
	{"reduce inputs", "reduce inputs as $i (0; . + $i)", ""},
}

func TestEdgeCaseReduceForeach(t *testing.T) {
	runEdgeCases(t, reduceForeachCases)
}

// ----- If / then / elif / else -----

var ifCases = []edgeCase{
	{"simple if/else", `if . == 0 then "zero" else "nonzero" end`, ""},
	{"if with elif", `if . == 0 then "zero" elif . == 1 then "one" else "many" end`, ""},
	{"multiple elif", `if . == 0 then "zero" elif . == 1 then "one" elif . == 2 then "two" else "many" end`, ""},
	{"if without else", "if . > 0 then . end", ""},
	{"if without else with elif", "if .a then .b elif .c then .d end", ""},
	{"if with pipe in condition", "if .a | .b then .c else .d end", ""},
	{"if with comma in branches", "if .x then .a, .b else .c, .d end", ""},
	{"nested if", "if .a then if .b then .c else .d end else .e end", ""},
	{"deeply nested if", `walk(if type == "array" then . else (if type == "object" then {name: .name?} else (if type == "string" then . else null end) end) end)`, ""},
	{"if with arithmetic in condition", `if (.a/.b) == 1 then "red" else "green" end`, ""},
	{"if with type check", `if type == "array" then .[] | .id elif type == "object" then .id else empty end`, ""},
	{"if in array", "[if .a then .b else .c end]", ""},
	{"if in object value", `{a: if .x then 1 else 2 end}`, ""},
	{"multiline if", "if . == 0 then\n  \"zero\"\nelif . == 1 then\n  \"one\"\nelse\n  \"many\"\nend", ""},
}

func TestEdgeCaseIf(t *testing.T) {
	runEdgeCases(t, ifCases)
}

// ----- Try / catch -----

var tryCases = []edgeCase{
	{"basic try", "try .a", ""},
	{"try catch", "try .a catch .", ""},
	{"try with pipe", "try .a | .b", ""},
	{"try catch with pipe", "try .a catch . | .b", ""},
	{"nested try", "try (try .a)", ""},
	{"try in array", "[.[] | try .a]", ""},
	{"try with error msg interp", `try error("invalid value: \(.)") catch .`, ""},
	{"try catch with string", `try .a catch ". is not an object"`, ""},
	{"try with comma body", "try .a, .b", ""},
	{"optional after division", "(1 / .)?", ""},
	{"optional after pipe", ".a | .b?", ""},
	{"optional after array", "[1, 2, 3]?", ""},
	{"tonumber optional", "[.[] | tonumber?]", ""},
	{"optional field in recursive", ".. | .a?", ""},
	{"try catch with if", `try repeat(exp) catch if .=="break" then empty else error end`, "parser limitation: catch handler doesn't support if expressions"},
	{"try with repeat error", `[repeat(.*2, error)?]`, ""},
}

func TestEdgeCaseTry(t *testing.T) {
	runEdgeCases(t, tryCases)
}

// ----- Operators and precedence -----

var operatorCases = []edgeCase{
	{"comma vs pipe", "true, false | not", ""},
	{"alternative vs comma", "false, 1 // 2", ""},
	{"mul vs add", "1 + 2 * 3", ""},
	{"comparison vs and", ".a > 0 and .b > 0", ""},
	{"parens override mul/add", "(1 + 2) * 3", ""},
	{"parens override comma/pipe", ".a | (.b, .c)", ""},
	{"complex precedence", ".a > 0 and .b > 0 or .c > 0", ""},
	{"chained pipe with comma", ".a | .b, .c | .d", ""},
	{"alternative with empty", "empty // 42", ""},
	{"generators with or", "(true, false) or false", ""},
	{"generators with and", "(true, true) and (true, false)", ""},
	{"not", "[true, false | not]", ""},
	{"chained alternatives", ".foo // .bar // .baz", ""},
	{"alt assign", ".foo //= .bar", ""},
}

func TestEdgeCaseOperators(t *testing.T) {
	runEdgeCases(t, operatorCases)
}

// ----- Assignment operators -----

var assignmentCases = []edgeCase{
	{"plain assign", ".foo = 42", ""},
	{"update assign", ".foo |= . + 1", ""},
	{"add assign", ".foo += 1", ""},
	{"sub assign", ".foo -= 1", ""},
	{"mul assign", ".foo *= 2", ""},
	{"div assign", ".foo /= 2", ""},
	{"mod assign", ".foo %= 2", ""},
	{"alt assign", ".foo //= 42", ""},
	{"assign with index", `.posts[0].title = "JQ Manual"`, ""},
	{"assign with iterator path", `.posts[].comments |= . + ["this is great"]`, ""},
	{"assign with pipe in path", `(.posts[] | select(.author == "stedolan") | .comments) |= . + ["terrible."]`, ""},
	{"assign with recursive", `(..|select(type=="boolean")) |= if . then 1 else 0 end`, ""},
	{"comma on lhs of assign", "(.a, .b) |= . + 1", ""},
	{"comma on lhs of plain assign", "(.a,.b) = range(2)", ""},
	{"comma on lhs of update assign", "(.a,.b) |= range(3)", ""},
	{"assign from field", ".a = .b", ""},
	{"update from field", ".a |= .b", ""},
	{"nested object with assign and comma", `{a:{b:{c:1}}} | (.a.b|=3), .`, ""},
}

func TestEdgeCaseAssignments(t *testing.T) {
	runEdgeCases(t, assignmentCases)
}

// ----- Objects -----

var objectCases = []edgeCase{
	{"empty object", "{}", ""},
	{"simple key value", "{a: 42}", ""},
	{"multiple keys", "{a: 42, b: 17}", ""},
	{"shorthand", "{foo}", ""},
	{"variable shorthand", "{$foo}", ""},
	{"quoted string key", `{"foo": 42}`, ""},
	{"expression key", `{("a"+"b"): 59}`, ""},
	{"variable key", "{$bar: $foo}", ""},
	{"pipe in expression key", `{(.a | .b): 1}`, ""},
	{"function call as key", `{(keys[0]): 1}`, ""},
	{"nested object", "{a: {b: {c: 1}}}", ""},
	{"object with array value", "{a: [1, 2, 3]}", ""},
	{"object with function call value", "{a: map(. + 1)}", ""},
	{"object with pipe in value", "{a: .x | .y}", ""},
	{"empty object then pipe", "{} | .", ""},
	{"object with if value", `{a: if .x then 1 else 2 end}`, ""},
	{"object with try value", "{a: try .x}", ""},
	{"object with reduce value", "{a: reduce .[] as $x (0; . + $x)}", ""},
	{"object with string key with spaces", `{"hello world": 1}`, ""},
	{"object with keyword key", `{true: 1, false: 0, null: 2}`, ""},
	{"object with nested object and array", `{a: {b: 1}, c: [2, 3]}`, ""},
	{"dynamic key from parenthesized", `{( .[0][-1] ): ( .[1] )}`, ""},
	{"variable key from machine", `. + { ($machine): {"total": (1 + $total)} }`, ""},
	{"object with walk", `map(. + {color:(if (.a/.b) == 1 then "red" else "green" end)})`, ""},
	{"with_entries", `with_entries(.key |= "KEY_" + .)`, ""},
	{"with_entries with sub", `walk( if type == "object" then with_entries( .key |= sub( "^_+"; "" ) ) else . end )`, ""},
}

func TestEdgeCaseObjects(t *testing.T) {
	runEdgeCases(t, objectCases)
}

// ----- Arrays -----

var arrayCases = []edgeCase{
	{"empty array", "[]", ""},
	{"array of numbers", "[1, 2, 3]", ""},
	{"array with pipe in element", "[.a | .b]", ""},
	{"array with iterator", "[.user, .projects[]]", ""},
	{"empty array then pipe", "[] | .", ""},
	{"array with if", "[if .a then .b else .c end]", ""},
	{"array with try", "[try .a]", ""},
	{"array with reduce", "[reduce .[] as $x (0; . + $x)]", ""},
	{"nested arrays", "[[1], [2, [3]]]", ""},
	{"array with comma in pipe", "[.a | .b, .c | .d]", ""},
	{"array of objects", "[{a: 1}, {b: 2}]", ""},
	{"array with select", `[.[] | select(.id == "second")]`, ""},
	{"array with tonumber optional", `[.[] | tonumber?]`, ""},
	{"array with map", `[.[] | .name]`, ""},
	{"array with limit", "[limit(3; .[])]", ""},
	{"array with while", "[while(.<100; .*2)]", ""},
	{"array with first", "first(.[])", ""},
}

func TestEdgeCaseArrays(t *testing.T) {
	runEdgeCases(t, arrayCases)
}

// ----- Comments -----

var commentCases = []edgeCase{
	{"trailing comment", ".foo # comment", ""},
	{"whole line comment", "# whole line comment\n.foo", ""},
	{"multi-line with comment", ".foo # comment\n| .bar", ""},
	{"comment in array", "[1, # comment\n 2]", ""},
	{"comment in object", "{a: # comment\n 1}", ""},
	{"comment in pipe", ".foo | # comment\n .bar", ""},
	{"comment in if", "if # comment\n .a then .b else .c end", ""},
	{"comment in def", "def f: # comment\n . + 1; f", ""},
	{"comment in reduce", "reduce .[] as $x ( # comment\n 0; . + $x)", ""},
	{"comment in function call", "map( # comment\n . + 1)", ""},
	{"comment in semicolon position", "reduce .[] as $x (0; # comment\n . + $x)", ""},
	{"comment after semicolon expr", "reduce .[] as $x (0; . + $x # comment\n )", ""},
	{"comment before else", "if .a then .b # comment\n else .c end", ""},
	{"mixed whitespace and comment", ".foo \n\t # comment\n | .bar", ""},
}

func TestEdgeCaseComments(t *testing.T) {
	runEdgeCases(t, commentCases)
}

// ----- String escapes -----

var stringEscapeCases = []edgeCase{
	{"newline escape", `"hello\nworld"`, ""},
	{"tab escape", `"tab\there"`, ""},
	{"quote escape", `"quote\"here"`, ""},
	{"backslash escape", `"backslash\\here"`, ""},
	{"unicode escape", `"\u0041"`, ""},
	{"unicode escape accented", `"\u00e9"`, ""},
	{"surrogate pair", `"\ud83d\ude00"`, ""},
	{"carriage return", `"\r"`, ""},
	{"formfeed", `"\f"`, ""},
	{"backspace", `"\b"`, ""},
	{"slash in string", `"a/b"`, ""},
	{"multiple escapes", `"line1\n\ttab"`, ""},
	{"control chars combined", `"\r\t\f\b"`, ""},
}

func TestEdgeCaseStringEscapes(t *testing.T) {
	runEdgeCases(t, stringEscapeCases)
}

// ----- Quoted field access -----

var quotedFieldCases = []edgeCase{
	{"dollar sign in field", `."foo$"`, ""},
	{"space in field", `."foo bar"`, ""},
	{"unicode field", `."日本語"`, ""},
	{"number field", `."123"`, ""},
	{"nested quoted fields", `.foo."bar".baz`, ""},
}

func TestEdgeCaseQuotedFields(t *testing.T) {
	runEdgeCases(t, quotedFieldCases)
}

// ----- Label / break -----

var labelCases = []edgeCase{
	{"label with reduce", "label $out | reduce .[] as $item (0; . + $item)", ""},
	{"label simple", "label $out | .", ""},
	{"label with break", "label $out | reduce .[] as $item (0; if . > 10 then break $out else . + $item end)", ""},
	{"label with false break", `label $out | reduce .[] as $item (null; if . == false then break $out else . + $item end)`, ""},
	{"nested labels", "label $out | label $inner | .", ""},
	{"break in nested label", "label $out | label $inner | if .a then break $inner else break $out end", ""},
}

func TestEdgeCaseLabel(t *testing.T) {
	runEdgeCases(t, labelCases)
}

// ----- Function calls -----

var functionCallCases = []edgeCase{
	{"no args", "keys", ""},
	{"one arg", "map(. + 1)", ""},
	{"two semicolon args", "limit(3; .[])", ""},
	{"three semicolon args", "range(0; 10; 3)", ""},
	{"nested function call", "map(select(. > 2))", ""},
	{"function with pipe in arg", "map(.a | .b)", ""},
	{"function with comma args", `any(.tags[]; .name == "TAG")`, ""},
	{"function with and in args", `any(.tags[]; .name == "TAG" and .x > 0)`, ""},
	{"any with generator", `map(select( any(.tags[]; .name == "TAG" )))`, ""},
	{"all with generator", `map(select( all( .tags[]; .name != "TAG") ))`, ""},
	{"INDEX", `INDEX(.[]; .id)`, ""},
	{"IN", `IN(.[]; 1, 2, 3)`, ""},
	{"JOIN", `JOIN($idx; .[]; .key; .value)`, ""},
	{"recurse no args", "recurse", ""},
	{"recurse with arg", "recurse(.children[])", ""},
	{"recurse with two args", "recurse(. * .; . < 20)", ""},
	{"until", `[.,1] | until(.[0] < 1; [.[0] - 1, .[1] * .[0]]) | .[1]`, ""},
	{"while", "[while(.<100; .*2)]", ""},
	{"debug", `debug("Entering function foo with $x == \($x)", .)`, ""},
	{"halt_error", "halt_error(1)", ""},
	{"error function", `error("something")`, ""},
	{"error with interp", `error("invalid value: \(.)")`, ""},
	{"empty", "empty", ""},
	{"inputs", "inputs", ""},
	{"input", "input", ""},
	{"env", "env", ""},
	{"$ENV", "$ENV", ""},
	{"$ARGS", "$ARGS", ""},
	{"del single", "del(.foo)", ""},
	{"del multiple paths", "del(.a, .b)", ""},
	{"del nested paths", "del(.a.b, .c)", ""},
	{"getpath", `getpath(["a","b"])`, ""},
	{"getpath multiple", `[getpath(["a","b"], ["a","c"])]`, ""},
	{"setpath", `setpath(["a","b"]; 1)`, ""},
	{"setpath with number key", `setpath([0,"a"]; 1)`, ""},
	{"delpaths", `delpaths([["a","b"]])`, ""},
	{"pick", "pick(.[2], .[0], .[0])", ""},
	{"path with index", "path(.a[0].b)", ""},
	{"path with recursive", `path(..|select(type=="boolean"))`, ""},
	{"paths with filter", `[paths(type == "number")]`, ""},
	{"path of paths", `[path(..)]`, ""},
}

func TestEdgeCaseFunctionCalls(t *testing.T) {
	runEdgeCases(t, functionCallCases)
}

// ----- Regex functions (sub/gsub/test/match/capture/scan/split) -----

var regexCases = []edgeCase{
	{"sub", `sub("^ +";"")`, ""},
	{"gsub", `gsub("(?<x>.)[^a]*"; "+\(.x)-")`, ""},
	{"match", `match("foo (?<bar123>bar)? foo"; "ig")`, ""},
	{"capture", `capture("(?<a>[a-z]+)-(?<n>[0-9]+)")`, ""},
	{"scan", `scan("(a+)(b+)")`, ""},
	{"split with null flag", `split(", *"; null)`, ""},
	{"splits", `splits(",? *"; "n")`, ""},
	{"sub with interp in replacement", `sub("[^a-z]*(?<x>[a-z]+)"; "Z\(.x)"; "g")`, ""},
	{"gsub with array replacement", `[gsub("p"; "a", "b")]`, ""},
	{"sub with pipe", `sub("^ +";"") | sub(" +$";"")`, ""},
	{"test with flags", `test("a b c # spaces are ignored"; "ix")`, ""},
	{"sort_by with scan and tonumber", `sort_by(.id|scan("[0-9]*$")|tonumber)`, ""},
}

func TestEdgeCaseRegex(t *testing.T) {
	runEdgeCases(t, regexCases)
}

// ----- Whitespace and formatting -----

var whitespaceCases = []edgeCase{
	{"lots of spaces", "  .foo   |   .bar  ", ""},
	{"newlines in pipe", ".foo\n|\n.bar", ""},
	{"tabs", ".foo\t|\t.bar", ""},
}

func TestEdgeCaseWhitespace(t *testing.T) {
	runEdgeCases(t, whitespaceCases)
}

// ----- Nested parentheses -----

var parenthesizedCases = []edgeCase{
	{"nested parens", "((.a))", ""},
	{"parens with comma", "(.a, (.b, .c))", ""},
	{"parens with def", "(def f: .; f)", ""},
	{"parens with reduce", "(reduce .[] as $x (0; . + $x))", ""},
	{"parenthesized binary", "(. + 2) * 5", ""},
	{"comma in parens then pipe", "(.foo, .bar) | .baz", ""},
}

func TestEdgeCaseParenthesized(t *testing.T) {
	runEdgeCases(t, parenthesizedCases)
}

// ----- Index and slice variations -----

var indexSliceCases = []edgeCase{
	{"index with number", ".[0]", ""},
	{"index with negative", ".[-1]", ""},
	{"index with string", `.["foo"]`, ""},
	{"index with variable", ".[$i]", ""},
	{"index with expression", ".[.a + .b]", ""},
	{"index with function call", ".[keys[0]]", ""},
	{"slice with numbers", ".[2:4]", ""},
	{"slice omit start", ".[:3]", ""},
	{"slice omit end", ".[-2:]", ""},
	{"slice negative both", ".[-3:-1]", ""},
	{"slice with expressions", ".[.a:.b]", ""},
	{"slice with variables", ".[$a:$b]", ""},
}

func TestEdgeCaseIndexSlice(t *testing.T) {
	runEdgeCases(t, indexSliceCases)
}

// ----- Known limitations (parser does not support these yet) -----

var knownLimitationCases = []edgeCase{
	// Comma inside bracket access: .[1, 2] is valid jq but the parser
	// currently uses parseExp (no comma) for bracket contents.
	{"comma in bracket access", "del(.[1, 2])", "parser limitation: comma inside bracket access not supported"},
	{"path with comma in bracket", "path(.[1, 2])", "parser limitation: comma inside bracket access not supported"},

	// Module system: import/include/module are explicitly not yet supported.
	{"include", `include "library"`, "parser limitation: module system not yet supported"},
	{"include with search", `include "sigma" {search: "~/jq"}`, "parser limitation: module system not yet supported"},
	{"import", `import "library" as lib`, "parser limitation: module system not yet supported"},
	{"import with symbol", `import "library" as lib; lib::walk(.)`, "parser limitation: module system not yet supported"},

	// .5 is not a number in jq — it's . followed by 5, which is invalid.
	// This is correct behavior: jq requires a leading zero (0.5).
	{"leading dot number", ".5", "not valid jq (requires 0.5)"},
}

func TestEdgeCaseKnownLimitations(t *testing.T) {
	runEdgeCases(t, knownLimitationCases)
}

// ----- Semicolon vs comma in function args -----
// In jq, ';' separates arguments and ',' creates a comma expression
// within a single argument.

var semicolonCommaCases = []edgeCase{
	{"semicolon args", "limit(3; .[])", ""},
	{"comma as single arg (del)", "del(.a, .b)", ""},
	{"comma as single arg (pick)", "pick(.[2], .[0], .[0])", ""},
	{"debug with comma expr", `debug("msg", .)`, ""},
	{"getpath with comma", `getpath(["a","b"], ["a","c"])`, ""},
	{"INDEX semicolon", "INDEX(.[]; .id)", ""},
	{"IN semicolon", "IN(.[]; 1, 2, 3)", ""},
	{"JOIN semicolon", "JOIN($idx; .[]; .key; .value)", ""},
	{"any with semicolon", `any(.tags[]; .name == "TAG")`, ""},
	{"all with semicolon", `all(.tags[]; .name != "TAG")`, ""},
	{"range three args", "range(0; 10; 3)", ""},
	{"limit two args", "limit(3; .[])", ""},
	{"split with semicolon", `split(", *"; null)`, ""},
	{"sub with semicolon", `sub("^ +";"")`, ""},
	{"def with semicolon args", "def f(a;b;c): a | b | c; f(.x; .y; .z)", ""},
	{"def with mixed args", "def foo(f; $v): f | . + $v; foo(.a; 5)", ""},
}

func TestEdgeCaseSemicolonComma(t *testing.T) {
	runEdgeCases(t, semicolonCommaCases)
}

// ----- Pipe right-associativity in sub-expressions -----
// Pipe is right-associative in all contexts, including arrays,
// objects, function args, and string interpolation.

var pipeAssocCases = []edgeCase{
	{"pipe right-assoc top level", ".a | .b | .c", ""},
	{"pipe right-assoc in array", "[.a | .b | .c]", ""},
	{"pipe right-assoc in object", "{x: .a | .b | .c}", ""},
	{"pipe right-assoc in function", "map(.a | .b | .c)", ""},
	{"pipe right-assoc in string interp", `"\(.a | .b | .c)"`, ""},
	{"pipe right-assoc in reduce", "reduce .[] as $x (0; . + $x | . * 2)", ""},
	{"pipe right-assoc in if", "if .a | .b then .c else .d end", ""},
	{"pipe right-assoc in try", "try .a catch .c", ""},
	{"as binding in function arg", "map(.x as $y | $y)", ""},
	{"pipe right-assoc in paren", "(.a | .b | .c)", ""},
}

func TestEdgeCasePipeAssoc(t *testing.T) {
	runEdgeCases(t, pipeAssocCases)
}

// ----- Keyword field names -----
// Keywords like if, then, else, end are valid field names in jq.

var keywordFieldCases = []edgeCase{
	{"field if", ".if", ""},
	{"field then", ".then", ""},
	{"field else", ".else", ""},
	{"field end", ".end", ""},
	{"field def", ".def", ""},
	{"field reduce", ".reduce", ""},
	{"field foreach", ".foreach", ""},
	{"field try", ".try", ""},
	{"field catch", ".catch", ""},
	{"field and", ".and", ""},
	{"field or", ".or", ""},
	{"field not", ".not", ""},
	{"field as", ".as", ""},
	{"field label", ".label", ""},
	{"field break", ".break", ""},
	{"field import", ".import", ""},
	{"field include", ".include", ""},
	{"field module", ".module", ""},
	{"keyword field then access", ".if.then", ""},
	{"keyword field in object", "{if: 1}", ""},
}

func TestEdgeCaseKeywordFields(t *testing.T) {
	runEdgeCases(t, keywordFieldCases)
}

// ----- Complex real-world programs from the jq Cookbook -----

var cookbookCases = []edgeCase{
	{"bag", "reduce .[] as $x ({}; .[$x|type][$x|tostring] += 1)", ""},
	{"bag function def", `def bag(stream): reduce stream as $x ({}; .[$x|type][$x|tostring] += 1); bag(.[])`, ""},
	{"maximal_by", `def maximal_by(f): (map(f) | max) as $mx | .[] | select(f == $mx); maximal_by(.x)`, ""},
	{"objectify full", `def objectify(headers): def tonumberq: tonumber? // .; def trimq: if type == "string" then sub("^ +";"") | sub(" +$";"") else . end; def tonullq: if . == "" then null else . end; . as $in | reduce range(0; headers|length) as $i ({}; .[headers[$i]] = ($in[$i] | trimq | tonumberq | tonullq)); objectify([])`, ""},
	{"uniq with foreach", `def uniq(s): foreach s as $x (null; if . == null or .emitted != $x then {emit: true, emitted: $x} else .emit = false end; if .emit then $x else empty end); uniq(.)`, ""},
	{"atomize", `def atomize(s): fromstream(foreach s as $in ({previous:null, emit: null}; if ($in | length == 2) and ($in|.[0][0]) != .previous and .previous != null then {emit: [[.previous]], previous: $in|.[0][0]} else { previous: ($in|.[0][0]), emit: null} end; (.emit // empty), $in)); atomize(.)`, ""},
	{"fromstream truncate", `fromstream(1|truncate_stream(inputs) | select(length>1) | .[0] |= .[1:])`, ""},
	{"tostream roundtrip", `. as $dot|fromstream($dot|tostream)|.==$dot`, ""},
	{"walk sort arrays", `walk(if type == "array" then sort else . end)`, ""},
	{"walk del keys", `walk(if type == "object" then del(.foo) else . end)`, ""},
	{"walk with_entries sub", `walk( if type == "object" then with_entries( .key |= sub( "^_+"; "" ) ) else . end )`, ""},
	{"recurse del", `recurse(.children[]) |= del(.foo)`, ""},
	{"recurse whitelist", `recurse(.children[]) |= {name, children}`, ""},
	{"template engine", `reduce paths as $p (.; getpath($p) as $v | if $v|type == "string" and $dict[0][$v] then setpath($p; $dict[0][$v]) else . end)`, ""},
	{"group_by reduce", `group_by(.keys) | reduce . as $x (.[]; add)`, ""},
	{"add keys union", `add | keys`, ""},
	{"max_by S3", `.Contents | max_by(.LastModified) | {Key}`, ""},
	{"zip headers", `(.columnHeaders | map(.name)) as $headers | .rows | map(. as $row | $headers | with_entries({"key": .value, "value": $row[.key]}))`, ""},
	{"select with any", `map(select( any(.tags[]; .name == "TAG" )))`, ""},
	{"select with all", `map(select( all( .tags[]; .name != "TAG") ))`, ""},
	{"select with any boolean", `map(select([ .tags[] | .name == "TAG" ] | any))`, ""},
	{"select with all boolean", `map(select([ .tags[] | .name != "TAG" ] | all))`, ""},
	{"select with test", `map( select(.genre | test("HOUSE"; "i")))`, ""},
	{"select with contains", `select(.genre | contains("house"))`, ""},
	{"select with contains and guard", `select(.genre | . and contains("house"))`, ""},
	{"color map", `map(. + {color:(if (.a/.b) == 1 then "red" else "green" end)})`, ""},
	{"color assign", `map(.color = if (.a/.b) == 1 then "red" else "green" end)`, ""},
	{"ncdu paths", `[.[3] | paths(scalars) as $p | [$p, getpath($p)] | { keys: .[0][0:-1], ( .[0][-1] ): ( .[1] ) }]`, ""},
	{"ncdu group_by", `group_by(.keys) | reduce . as $x (.[]; add)`, ""},
	{"map_values alt", `map_values(. // empty)`, ""},
	{"fromjson", `.key|fromjson`, ""},
	{"reduce large input", `reduce inputs as $line ({}; $line.machine as $machine | .[$machine].total as $total | . + { ($machine): {"total": (1 + $total)} })`, ""},
	{"foreach extract", `foreach (inputs, null) as $line (0; if $line.n then .+1 else . end; if $line == null then . else $line.n // empty end)`, ""},
}

func TestEdgeCaseCookbook(t *testing.T) {
	runEdgeCases(t, cookbookCases)
}

// ----- Error cases (should produce parse errors) -----

func TestEdgeCaseErrorCases(t *testing.T) {
	errorCases := []string{
		"",                        // empty input
		".foo |",                  // trailing pipe
		".foo .",                  // double dot
		"if . > 0 then .",         // incomplete if
		"reduce .[] as $item (0)", // incomplete reduce
		"[1, 2",                   // unclosed array
		"{a: ",                    // incomplete object
		"def foo: .",              // incomplete def
		"break",                   // break without label
		`"unterminated`,           // unclosed string
		".foo .bar .baz",          // consecutive fields without dot
		"if then else end",        // if without condition
		"try catch .",             // try without body
		"reduce .[] as (0; .)",    // reduce without variable
		"foreach .[] as $x (0)",   // foreach missing semicolon
		"label $out",              // label without pipe
		`{"unterminated`,          // unclosed object string key
		"[1, 2, 3, ]",             // trailing comma in array
		"{a: 1, }",                // trailing comma in object
		// Non-associative operators (jq rejects chaining)
		".a == .b == .c",          // comparison non-assoc
		".a < .b < .c",            // comparison non-assoc
		".a > .b > .c",            // comparison non-assoc
		".a != .b != .c",          // comparison non-assoc
		".a <= .b <= .c",          // comparison non-assoc
		".a >= .b >= .c",          // comparison non-assoc
		".a = .b = .c",            // assignment non-assoc
		".a |= .b |= .c",          // update assignment non-assoc
		".a += .b += .c",          // update assignment non-assoc
		".a -= .b -= .c",          // update assignment non-assoc
		".a *= .b *= .c",          // update assignment non-assoc
		".a /= .b /= .c",          // update assignment non-assoc
		".a %= .b %= .c",          // update assignment non-assoc
		".a //= .b //= .c",        // update assignment non-assoc
		// Empty destructuring patterns (jq rejects)
		". as {} | .",             // empty object pattern
		". as [] | .",             // empty array pattern
		// Comment-only input
		"# comment only",          // no expression, just comment
		// Bare @ without format name
		"@",                       // @ needs a format name
		// Adjacent format strings without pipe
		"@base64 @text",           // need pipe between formats
	}
	for _, input := range errorCases {
		t.Run("error:"+input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("parse %q: expected error, got nil", input)
			}
		})
	}
}
