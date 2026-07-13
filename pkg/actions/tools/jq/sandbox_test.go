package jq

import (
	"strings"
	"testing"

	"github.com/carapace-sh/carapace"
	jqparser "github.com/carapace-sh/carapace-jq/pkg/jq"
	"github.com/carapace-sh/carapace/pkg/sandbox"
)

// Sandbox tests for ActionFilters at various cursor positions within
// complex jq expressions.  Tests use Expect (exact match) to verify
// that correct completions are returned, built from the same base
// action functions (ActionBuiltins, ActionKeywords, etc.) that
// ActionFilters uses internally.
//
// The sandbox strips Uids in comparisons, so actions built directly
// from base functions match those roundtripped through Invoke().ToA()
// inside ActionFilters.

// expressionAction returns the full expression-context action:
// builtins + keywords + literals + special filters + formats.
func expressionAction() carapace.Action {
	return carapace.Batch(
		ActionBuiltins(),
		ActionKeywords(),
		ActionLiterals(),
		ActionSpecialFilters(),
		ActionFormats(),
	).ToA()
}

// prefixAndToken splits an input into typedPrefix and partialToken,
// mirroring ActionFilters' internal logic.
func prefixAndToken(input string) (typedPrefix, partialToken string) {
	typedPrefix = ""
	partialToken = input
	if lastSpace := strings.LastIndex(input, " "); lastSpace >= 0 {
		typedPrefix = input[:lastSpace+1]
		partialToken = input[lastSpace+1:]
	}

	ctx := jqparser.ParseForCompletion(input)
	if !strings.Contains(input, " ") && !hasExpected(ctx, jqparser.ExpectedExpression) && !hasExpected(ctx, jqparser.ExpectedFormatName) {
		typedPrefix = input
		partialToken = ""
	}
	return
}

// expectExpression runs ActionFilters on input and asserts the output
// equals expressionAction with the correct prefix applied.
func expectExpression(t *testing.T, s *sandbox.Sandbox, input string) {
	t.Helper()
	typedPrefix, _ := prefixAndToken(input)
	expected := expressionAction()
	if typedPrefix != "" {
		expected = expected.Prefix(typedPrefix)
	}
	s.Run(input).Expect(expected)
}

// ----- Start of expression -----

func TestSandboxEmpty(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "")
	})
}

func TestSandboxAfterPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".foo | ")
	})
}

func TestSandboxAfterArithmetic(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".a + ")
	})
}

func TestSandboxAfterAlternative(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".foo // ")
	})
}

func TestSandboxAfterAnd(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".a and ")
	})
}

func TestSandboxAfterOr(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".a or ")
	})
}

func TestSandboxAfterComparison(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".a > ")
	})
}

func TestSandboxAfterNegate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "-")
	})
}

func TestSandboxAfterIf(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "if ")
	})
}

func TestSandboxAfterThen(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "if . > 0 then ")
	})
}

func TestSandboxAfterTry(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "try ")
	})
}

func TestSandboxAfterLabel(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "label $out | ")
	})
}

func TestSandboxAfterDefBody(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "def myfunc: . + 1; ")
	})
}

func TestSandboxNestedDefRestoresContext(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "def f: def g: .; g; ")
	})
}

func TestSandboxDefWithArgsComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "def map(f): [.[] | f]; ")
	})
}

func TestSandboxInObjectAfterColon(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "{foo: ")
	})
}

func TestSandboxObjectValueWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "{foo: .a | ")
	})
}

func TestSandboxComplexObjectWithExprKey(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, `{("a"+"b"): `)
	})
}

func TestSandboxInReduceInit(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "reduce .[] as $item ( ")
	})
}

func TestSandboxInReduceUpdate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "reduce .[] as $item (0; ")
	})
}

func TestSandboxInForeachUpdate(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "foreach .[] as $item (0; ")
	})
}

func TestSandboxReduceSourceWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "reduce .a | .b as $x ( ")
	})
}

func TestSandboxForeachSourceWithPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "foreach .a | .b as $x ( ")
	})
}

func TestSandboxInStringInterp(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, `"hello \( `)
	})
}

func TestSandboxDestructuringComplete(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ". as [$a, $b] | ")
	})
}

func TestSandboxComplexPipeChain(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".realnames as $names | .posts[] | {title, author: $names[.author]} | ")
	})
}

func TestSandboxComplexNestedFunction(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "map(select(. > 2) | ")
	})
}

func TestSandboxComplexReduce(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "reduce .[] as $item (0; . + $item | ")
	})
}

func TestSandboxComplexRecursiveDescent(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".. | .a? | ")
	})
}

func TestSandboxComplexNestedDefs(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while; ")
	})
}

func TestSandboxComplexForeachExtract(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "foreach .[] as $item (0; . + $item; [ ")
	})
}

func TestSandboxCommentInPipe(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, ".foo | # comment\n ")
	})
}

func TestSandboxCommentBeforeExpression(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "# whole line comment\n ")
	})
}

func TestSandboxInArrayAfterComma(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "[1, ")
	})
}

func TestSandboxArrayWithPipeInElement(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "[.a | ")
	})
}

func TestSandboxInFunctionCall(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// Function context returns expressionAction only (no closing paren)
		expectExpression(t, s, "map( ")
	})
}

func TestSandboxInFunctionAfterComma(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "limit(3, ")
	})
}

func TestSandboxInFunctionSemicolonArg(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "limit(3; ")
	})
}

func TestSandboxFunctionWithPipeInArg(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "map(.a | ")
	})
}

// ----- Format name -----

func TestSandboxFormatName(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("@").Expect(ActionFormats())
	})
}

// ----- Operator context -----

func TestSandboxOperatorContext(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion(".foo ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(".foo ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(".foo "),
		)
	})
}

func TestSandboxAfterIdentityOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion(". ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(". ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(". "),
		)
	})
}

func TestSandboxIfCompleteOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("if . > 0 then . else . end ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run("if . > 0 then . else . end ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix("if . > 0 then . else . end "),
		)
	})
}

func TestSandboxTryCatchCompleteOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion(`try .a catch . `)
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(`try .a catch . `).Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(`try .a catch . `),
		)
	})
}

func TestSandboxReduceCompleteOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("reduce .[] as $item (0; . + $item) ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run("reduce .[] as $item (0; . + $item) ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix("reduce .[] as $item (0; . + $item) "),
		)
	})
}

func TestSandboxComplexWalkOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, `walk(if type == "array" then sort else . end) `)
	})
}

func TestSandboxComplexAssignmentOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		input := `.posts[].comments |= . + ["this is great"] `
		ctx := jqparser.ParseForCompletion(input)
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(input).Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(input),
		)
	})
}

func TestSandboxPostfixAfterArrayOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		input := "[1, 2, 3] "
		ctx := jqparser.ParseForCompletion(input)
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(input).Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(input),
		)
	})
}

func TestSandboxPostfixAfterStringOperator(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		input := `"hello" `
		ctx := jqparser.ParseForCompletion(input)
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		s.Run(input).Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
			).ToA().Prefix(input),
		)
	})
}

// ----- Special contexts (non-expression, non-operator) -----

func TestSandboxInObject(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("{").Expect(carapace.ActionMessage("object key"))
	})
}

func TestSandboxAfterDot(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run(".foo.").Expect(carapace.ActionMessage("field name"))
	})
}

func TestSandboxInDef(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("def ").Expect(carapace.ActionMessage("function name"))
	})
}

func TestSandboxAfterBreak(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("break ").Expect(carapace.ActionValues("$").NoSpace().Prefix("break "))
	})
}

func TestSandboxAsBindingPattern(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		// ".foo as $" — $ already typed, parser expects pipe after pattern
		// partialToken="$", action=ActionValues("|").NoSpace(), prefix=".foo as "
		// FilterPrefix(".foo as $") removes "|" since ".foo as |" doesn't start with ".foo as $"
		// So actual is empty — verify it matches the pipe action (not expression values)
		s.Run(".foo as $").Expect(carapace.ActionValues("|").NoSpace().Prefix(".foo as "))
	})
}

func TestSandboxPatternAlternativeAtCursor(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run(".foo as [$a] ?// ").Expect(carapace.ActionValues("$").NoSpace().Prefix(".foo as [$a] ?// "))
	})
}

func TestSandboxAfterFormat(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "@base64 ")
	})
}

func TestSandboxPartialString(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run(`"fo`).Expect(carapace.ActionValues(`"`).NoSpace().Prefix(`"fo`))
	})
}

// ----- Keyword context -----

func TestSandboxAfterTryBody(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("try .a ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		keywordTokens := actionForKeywordTokens(ctx)
		s.Run("try .a ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
				keywordTokens,
			).ToA().Prefix("try .a "),
		)
	})
}

func TestSandboxIfConditionExpectsThen(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("if .a ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		keywordTokens := actionForKeywordTokens(ctx)
		s.Run("if .a ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
				keywordTokens,
			).ToA().Prefix("if .a "),
		)
	})
}

func TestSandboxIfThenBodyContext(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("if . > 0 then . ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		keywordTokens := actionForKeywordTokens(ctx)
		s.Run("if . > 0 then . ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
				keywordTokens,
			).ToA().Prefix("if . > 0 then . "),
		)
	})
}

func TestSandboxIfExpectingEnd(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		ctx := jqparser.ParseForCompletion("if . > 0 then . else . ")
		ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
		for _, op := range ctx.ValidOperators {
			ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
		}
		keywordTokens := actionForKeywordTokens(ctx)
		s.Run("if . > 0 then . else . ").Expect(
			carapace.Batch(
				ActionOperators(ops).NoSpace(),
				keywordTokens,
			).ToA().Prefix("if . > 0 then . else . "),
		)
	})
}

func TestSandboxAfterElse(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		expectExpression(t, s, "if . > 0 then . else ")
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
