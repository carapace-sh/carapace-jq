package jq

import (
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/sandbox"
)

// Sandbox tests for ActionFilters at various cursor positions within
// complex jq expressions.  Each test feeds a partial expression to
// ActionFilters and asserts on the completion output using ExpectNot
// (the proven pattern from the existing tests).
//
// The sandbox.Action wrapper generates UIDs dynamically inside
// ActionFilters' callback, so Expect (exact match) is unreliable —
// we use ExpectNot to verify that certain values are present or absent.
//
// To assert "expression context" we verify that expression-context
// values (builtins like "map(") ARE present by using ExpectNot with
// an action that contains only operator-context values (like "|").
// If the actual output equals the operator-only action, the test
// fails — proving expression values are present.
//
// To assert "operator context" we verify that expression-context
// values are NOT present by using ExpectNot with the builtins directly.
//
// Expression positions tested:
//   - Empty / start of expression
//   - After identity (.)
//   - After field access (.foo)
//   - After dot (awaiting field name)
//   - After pipe (awaiting expression)
//   - After operators (+, //, and, or, ==)
//   - After negate (-)
//   - Inside function call (map(, select(, limit(3, )
//   - Inside function after comma (limit(3, )
//   - Inside function after semicolon (limit(3; )
//   - Inside array construction ([, [1, )
//   - Inside object construction ({, {foo: )
//   - Inside if/then/elif/else/end
//   - Inside try / catch
//   - Inside reduce/foreach (init, update, extract)
//   - Inside def (name, body, rest)
//   - Inside string interpolation ("hello \()
//   - Inside format name (@)
//   - After label (label $out |)
//   - In as-pattern (.foo as $)
//   - After slice colon (.[2:)
//   - At closing brackets/parens/braces
//   - At keyword positions (if .a , try .a )
//   - Complex multi-position expressions (pipe chains, nested calls)

// expressionValues is a set of values that only appear in expression
// context (when the parser expects a new primary expression).
// Used with ExpectNot: if the actual output equals this action, the
// cursor is NOT in expression context (test fails).
var expressionValues = carapace.ActionValues("map(", "keys()", "length()", "true", "false", "null", ".", "..", "@base64")

// operatorValues is a set of values that only appear in operator
// context (when the parser expects an infix/postfix operator).
// Used with ExpectNot: if the actual output equals this action, the
// cursor is NOT in operator context (test fails).
var operatorValues = carapace.ActionValues("|", ",", "//", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "and", "or")

// assertExpressionContext verifies that the completion output contains
// expression-context values (builtins, literals, special filters).
// It does this by asserting the output differs from an operator-only action.
func assertExpressionContext(t *testing.T, s *sandbox.Sandbox, input string) {
	t.Helper()
	s.Run(input).ExpectNot(operatorValues)
}

// assertOperatorContext verifies that the completion output does NOT
// contain expression-context values (builtins, literals).
func assertOperatorContext(t *testing.T, s *sandbox.Sandbox, input string) {
	t.Helper()
	s.Run(input).ExpectNot(expressionValues)
}

// ----- Start of expression -----

func TestSandboxEmpty(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "")
	})
}

func TestSandboxPartialBuiltin(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "ma")
	})
}

func TestSandboxPartialKeyword(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "re")
	})
}

// ----- After identity / field access -----

func TestSandboxAfterIdentity(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, ".")
	})
}

func TestSandboxAfterField(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, ".foo")
	})
}

func TestSandboxAfterDot(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// AfterDot — field name context, not expression
		assertOperatorContext(t, s, ".foo.")
	})
}

// ----- After pipe -----

func TestSandboxAfterPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".foo | ")
	})
}

func TestSandboxAfterPipePartial(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".foo | ke")
	})
}

// ----- After operators -----

func TestSandboxAfterArithmetic(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".a + ")
	})
}

func TestSandboxAfterAlternative(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".foo // ")
	})
}

func TestSandboxAfterAnd(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".a and ")
	})
}

func TestSandboxAfterOr(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".a or ")
	})
}

func TestSandboxAfterComparison(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".a > ")
	})
}

func TestSandboxAfterNegate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "-")
	})
}

// ----- Inside function calls -----

func TestSandboxInFunctionCall(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "map(")
	})
}

func TestSandboxInFunctionAfterComma(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "limit(3, ")
	})
}

func TestSandboxInFunctionSemicolonArg(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "limit(3; ")
	})
}

func TestSandboxNestedFunctionCall(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "map(select( ")
	})
}

func TestSandboxFunctionWithPipeInArg(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "map(.a | ")
	})
}

// ----- Inside array construction -----

func TestSandboxInArray(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "[ ")
	})
}

func TestSandboxInArrayAfterComma(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "[1, ")
	})
}

func TestSandboxArrayWithPipeInElement(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "[.a | ")
	})
}

// ----- Inside object construction -----

func TestSandboxInObject(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Object key context — not expression, not operator
		assertOperatorContext(t, s, "{")
	})
}

func TestSandboxInObjectAfterColon(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "{foo: ")
	})
}

func TestSandboxObjectValueWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "{foo: .a | ")
	})
}

// ----- Inside if/then/elif/else/end -----

func TestSandboxAfterIf(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "if ")
	})
}

func TestSandboxAfterThen(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "if . > 0 then ")
	})
}

func TestSandboxAfterElse(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After else: expression or keyword "end" — not pure expression
		s.Run("if . > 0 then . else ").ExpectNot(operatorValues)
	})
}

func TestSandboxIfExpectingEnd(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After else body: operator or keyword "end" — not expression
		assertOperatorContext(t, s, "if . > 0 then . else . ")
	})
}

func TestSandboxIfConditionExpectsThen(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After if condition: operator + keyword "then" — not expression
		assertOperatorContext(t, s, "if .a ")
	})
}

func TestSandboxIfThenBodyContext(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After then body: keyword (elif, else, end) + operators — not expression
		assertOperatorContext(t, s, "if . > 0 then . ")
	})
}

func TestSandboxElifWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After complete if: operator context
		assertOperatorContext(t, s, "if .a then .b elif .c | .d then .e end ")
	})
}

func TestSandboxIfComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "if . > 0 then . else . end ")
	})
}

// ----- Inside try / catch -----

func TestSandboxAfterTry(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "try ")
	})
}

func TestSandboxAfterTryBody(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After try body: operator + keyword "catch" — not expression
		assertOperatorContext(t, s, "try .a ")
	})
}

func TestSandboxTryCatchComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "try .a catch . ")
	})
}

// ----- Inside reduce / foreach -----

func TestSandboxInReduceInit(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "reduce .[] as $item ( ")
	})
}

func TestSandboxInReduceUpdate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "reduce .[] as $item (0; ")
	})
}

func TestSandboxInForeachUpdate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "foreach .[] as $item (0; ")
	})
}

func TestSandboxReduceSourceWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "reduce .a | .b as $x ( ")
	})
}

func TestSandboxForeachSourceWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "foreach .a | .b as $x ( ")
	})
}

func TestSandboxReduceComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "reduce .[] as $item (0; . + $item) ")
	})
}

func TestSandboxForeachComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "foreach .[] as $item (0; . + $item; [$item, . * 2]) ")
	})
}

// ----- Inside def -----

func TestSandboxInDef(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Def name context — not expression, not operator
		assertOperatorContext(t, s, "def ")
	})
}

func TestSandboxAfterDefName(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After def name: expecting colon — not expression, not operator
		assertOperatorContext(t, s, "def myfunc")
	})
}

func TestSandboxAfterDefBody(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After def body semicolon: expression context (rest of program)
		assertExpressionContext(t, s, "def myfunc: . + 1; ")
	})
}

func TestSandboxNestedDefRestoresContext(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After nested def: expression context (rest of program, not def)
		assertExpressionContext(t, s, "def f: def g: .; g; ")
	})
}

func TestSandboxDefWithArgsComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "def map(f): [.[] | f]; ")
	})
}

// ----- Inside string interpolation -----

func TestSandboxInStringInterp(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, `"hello \( `)
	})
}

func TestSandboxStringInterpWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After complete string with interp: operator context
		assertOperatorContext(t, s, `"hello \(.a | .b)"`)
	})
}

func TestSandboxPartialString(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Partial string: string close context — not expression, not operator
		assertOperatorContext(t, s, `"fo`)
	})
}

// ----- Inside format name -----

func TestSandboxFormatName(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// "@" → format name context, not expression, not operator
		assertOperatorContext(t, s, "@")
	})
}

func TestSandboxAfterFormat(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After format: operator context
		assertOperatorContext(t, s, "@base64 ")
	})
}

func TestSandboxFormatWithInterpComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `@uri "https://example.com/search?q=\(.search)" `)
	})
}

// ----- Label / break -----

func TestSandboxAfterLabel(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "label $out | ")
	})
}

func TestSandboxAfterBreak(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After break: expecting $ — not expression, not operator
		assertOperatorContext(t, s, "break ")
	})
}

func TestSandboxLabelBreakComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "label $out | reduce .[] as $item (0; if . > 10 then break $out else . + $item end) ")
	})
}

// ----- As binding / destructuring -----

func TestSandboxAsBindingPattern(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// In as pattern: expecting $ — not expression, not operator
		assertOperatorContext(t, s, ".foo as $")
	})
}

func TestSandboxPatternAlternativeAtCursor(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After ?//: expecting $ — not expression, not operator
		assertOperatorContext(t, s, ".foo as [$a] ?// ")
	})
}

func TestSandboxDestructuringComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ". as [$a, $b] | ")
	})
}

// ----- Slice context -----

func TestSandboxAfterSliceStart(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After slice colon: expecting ] — not expression, not operator
		assertOperatorContext(t, s, ".[2:")
	})
}

// ----- Bracket access -----

func TestSandboxBracketAccessWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// After bracket with pipe inside: operator context
		assertOperatorContext(t, s, ".[.a | .b] ")
	})
}

// ----- After postfix on various bases -----

func TestSandboxPostfixAfterParen(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "(.a, .b) ")
	})
}

func TestSandboxPostfixAfterArray(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "[1, 2, 3] ")
	})
}

func TestSandboxPostfixAfterString(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `"hello" `)
	})
}

func TestSandboxPostfixAfterNumber(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "42 ")
	})
}

func TestSandboxPostfixAfterBool(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "true ")
	})
}

func TestSandboxPostfixAfterNull(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "null ")
	})
}

func TestSandboxPostfixAfterVariable(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "$foo ")
	})
}

func TestSandboxPostfixAfterRecursive(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, ".. ")
	})
}

// ----- Complex multi-position expressions -----

func TestSandboxComplexPipeChain(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".realnames as $names | .posts[] | {title, author: $names[.author]} | ")
	})
}

func TestSandboxComplexNestedFunction(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "map(select(. > 2) | ")
	})
}

func TestSandboxComplexReduce(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "reduce .[] as $item (0; . + $item | ")
	})
}

func TestSandboxComplexIfElif(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run(`if . == 0 then "zero" elif . == 1 then "one" else `).ExpectNot(operatorValues)
	})
}

func TestSandboxComplexDefWithArgs(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "def map(f): [.[] | f]; ")
	})
}

func TestSandboxComplexWalkWithIf(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `walk(if type == "array" then sort else . end) `)
	})
}

func TestSandboxComplexObjectWithExprKey(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, `{("a"+"b"): `)
	})
}

func TestSandboxComplexStringWithMultipleInterp(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `"The input was \(.), which is one less than \(.+1)" `)
	})
}

func TestSandboxComplexTryCatch(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `try .a catch ". is not an object" `)
	})
}

func TestSandboxComplexAssignment(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, `.posts[].comments |= . + ["this is great"] `)
	})
}

func TestSandboxComplexRecursiveDescent(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".. | .a? | ")
	})
}

func TestSandboxComplexForeachExtract(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "foreach .[] as $item (0; . + $item; [ ")
	})
}

func TestSandboxComplexNestedDefs(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while; ")
	})
}

func TestSandboxComplexObjectShorthand(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, ".realnames as $names | .posts[] | {title, author: $names[.author]} ")
	})
}

func TestSandboxComplexDelFunction(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "del(.foo) ")
	})
}

func TestSandboxComplexMapValues(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertOperatorContext(t, s, "map(. + 1) ")
	})
}

// ----- Comments -----

func TestSandboxCommentInPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, ".foo | # comment\n ")
	})
}

func TestSandboxCommentBeforeExpression(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		assertExpressionContext(t, s, "# whole line comment\n ")
	})
}

// ----- No double-at format prefix (regression) -----

func TestSandboxFormatNoDoubleAt(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("").ExpectNot(carapace.ActionValues(
			"@@text", "@@json", "@@html", "@@uri",
			"@@csv", "@@tsv", "@@sh", "@@base64", "@@base64d",
		))
		s.Run("@").ExpectNot(carapace.ActionValues(
			"@@text", "@@json", "@@html", "@@uri",
			"@@csv", "@@tsv", "@@sh", "@@base64", "@@base64d",
		))
	})
}

// ----- Multiple cursor positions in one expression -----

func TestSandboxMultiplePositions(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Position 1: after "map(" → expression context
		assertExpressionContext(t, s, "map(")

		// Position 2: after "map(. | " → expression after pipe in function
		assertExpressionContext(t, s, "map(. | ")

		// Position 3: after "map(. | .) " → operator context (function complete)
		assertOperatorContext(t, s, "map(. | .) ")
	})
}

func TestSandboxMultiplePositionsIf(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Position 1: "if " → expression context for condition
		assertExpressionContext(t, s, "if ")

		// Position 2: "if .a " → operator + keyword "then"
		assertOperatorContext(t, s, "if .a ")

		// Position 3: "if .a then " → expression context for then body
		assertExpressionContext(t, s, "if .a then ")

		// Position 4: "if .a then .b " → keyword (elif, else, end) + operators
		assertOperatorContext(t, s, "if .a then .b ")

		// Position 5: "if .a then .b else " → expression or keyword "end"
		s.Run("if .a then .b else ").ExpectNot(operatorValues)

		// Position 6: "if .a then .b else .c " → keyword "end" + operators
		assertOperatorContext(t, s, "if .a then .b else .c ")

		// Position 7: "if .a then .b else .c end " → operator context
		assertOperatorContext(t, s, "if .a then .b else .c end ")
	})
}

func TestSandboxMultiplePositionsReduce(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Position 1: "reduce " → expression context for source
		assertExpressionContext(t, s, "reduce ")

		// Position 2: "reduce .[] as $item ( " → expression context for init
		assertExpressionContext(t, s, "reduce .[] as $item ( ")

		// Position 3: "reduce .[] as $item (0; " → expression context for update
		assertExpressionContext(t, s, "reduce .[] as $item (0; ")

		// Position 4: "reduce .[] as $item (0; . + $item) " → operator context
		assertOperatorContext(t, s, "reduce .[] as $item (0; . + $item) ")
	})
}

func TestSandboxMultiplePositionsObject(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Position 1: "{" → object key context
		assertOperatorContext(t, s, "{")

		// Position 2: "{foo: " → expression context for value
		assertExpressionContext(t, s, "{foo: ")

		// Position 3: "{foo: .a " → operator + closing brace
		assertOperatorContext(t, s, "{foo: .a ")

		// Position 4: "{foo: .a, " → object key context (next entry)
		assertOperatorContext(t, s, "{foo: .a, ")

		// Position 5: "{foo: .a, bar: " → expression context for value
		assertExpressionContext(t, s, "{foo: .a, bar: ")
	})
}
