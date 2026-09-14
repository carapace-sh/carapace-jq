package jq

import (
	"slices"
	"testing"
)

func assertHasExpected(t *testing.T, ctx *CompletionContext, tok ExpectedToken) {
	t.Helper()
	if slices.Contains(ctx.ExpectedTokens, tok) {
		return
	}
	t.Errorf("expected token %v not found in %v", tok, ctx.ExpectedTokens)
}

func assertNotExpected(t *testing.T, ctx *CompletionContext, tok ExpectedToken) {
	t.Helper()
	if slices.Contains(ctx.ExpectedTokens, tok) {
		t.Errorf("token %v should not be in %v", tok, ctx.ExpectedTokens)
		return
	}
}

func assertHasOperator(t *testing.T, ctx *CompletionContext, op string) {
	t.Helper()
	for _, o := range ctx.ValidOperators {
		if o.Op == op {
			return
		}
	}
	t.Errorf("expected operator %q not found in %v", op, ctx.ValidOperators)
}

func TestCompletionEmpty(t *testing.T) {
	ctx := ParseForCompletion("")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterIdentity(t *testing.T) {
	ctx := ParseForCompletion(".")
	assertHasExpected(t, ctx, ExpectedOperator)
	assertHasOperator(t, ctx, "|")
	assertHasOperator(t, ctx, ",")
	assertHasOperator(t, ctx, "+")
}

func TestCompletionAfterField(t *testing.T) {
	ctx := ParseForCompletion(".foo")
	assertHasExpected(t, ctx, ExpectedOperator)
	if ctx.PartialIdent != "foo" {
		t.Errorf("expected PartialIdent 'foo', got %q", ctx.PartialIdent)
	}
}

func TestCompletionAfterDot(t *testing.T) {
	ctx := ParseForCompletion(".foo.")
	if !ctx.AfterDot {
		t.Error("expected AfterDot to be true")
	}
}

func TestCompletionAfterPipe(t *testing.T) {
	ctx := ParseForCompletion(".foo | ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterOperator(t *testing.T) {
	ctx := ParseForCompletion(".a + ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionPartialIdentifier(t *testing.T) {
	ctx := ParseForCompletion("ma")
	if ctx.PartialIdent != "ma" {
		t.Errorf("expected PartialIdent 'ma', got %q", ctx.PartialIdent)
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionInFunction(t *testing.T) {
	ctx := ParseForCompletion("map(")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingParen)
	if ctx.Function == nil {
		t.Fatal("expected Function context")
	}
	if ctx.Function.Name != "map" {
		t.Errorf("expected function 'map', got %q", ctx.Function.Name)
	}
	if ctx.Function.ArgIndex != 0 {
		t.Errorf("expected arg index 0, got %d", ctx.Function.ArgIndex)
	}
}

func TestCompletionInFunctionAfterArg(t *testing.T) {
	ctx := ParseForCompletion("map(.x")
	if ctx.Function == nil {
		t.Fatal("expected Function context")
	}
	if ctx.Function.Name != "map" {
		t.Errorf("expected function 'map', got %q", ctx.Function.Name)
	}
}

func TestCompletionInFunctionAfterComma(t *testing.T) {
	ctx := ParseForCompletion("limit(3; ")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingParen)
	if ctx.Function == nil {
		t.Fatal("expected Function context")
	}
	if ctx.Function.ArgIndex != 1 {
		t.Errorf("expected arg index 1, got %d", ctx.Function.ArgIndex)
	}
}

func TestCompletionInArray(t *testing.T) {
	ctx := ParseForCompletion("[")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingBracket)
}

func TestCompletionInArrayAfterElement(t *testing.T) {
	ctx := ParseForCompletion("[1, ")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingBracket)
}

func TestCompletionInObject(t *testing.T) {
	ctx := ParseForCompletion("{")
	if ctx.Object == nil {
		t.Fatal("expected Object context")
	}
	if !ctx.Object.InKey {
		t.Error("expected InKey to be true")
	}
}

func TestCompletionInObjectAfterKey(t *testing.T) {
	ctx := ParseForCompletion("{foo")
	if ctx.Object == nil {
		t.Fatal("expected Object context")
	}
}

func TestCompletionInObjectAfterColon(t *testing.T) {
	ctx := ParseForCompletion("{foo: ")
	if ctx.Object == nil {
		t.Fatal("expected Object context")
	}
	if !ctx.Object.InValue {
		t.Error("expected InValue to be true")
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterIf(t *testing.T) {
	ctx := ParseForCompletion("if ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterThen(t *testing.T) {
	ctx := ParseForCompletion("if . > 0 then ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterElse(t *testing.T) {
	ctx := ParseForCompletion("if . > 0 then . else ")
	// After 'else', an expression is expected (and 'end' keyword after it)
	// The completion context may include both expression and operator tokens
	// since the expression could be empty
	found := false
	for _, e := range ctx.ExpectedTokens {
		if e == ExpectedExpression || e == ExpectedKeyword {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Expression or Keyword, got %v", ctx.ExpectedTokens)
	}
}

func TestCompletionIfExpectingEnd(t *testing.T) {
	ctx := ParseForCompletion("if . > 0 then . else . ")
	// After the else body, 'end' keyword is expected, but also operators
	// since the expression could continue
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionIfConditionExpectsThen(t *testing.T) {
	ctx := ParseForCompletion("if .a ")
	assertHasExpected(t, ctx, ExpectedKeyword)
	assertHasExpected(t, ctx, ExpectedOperator)
	if ctx.If == nil || ctx.If.Section != "condition" {
		t.Errorf("expected If context with section 'condition', got %v", ctx.If)
	}
	if !slices.Contains(ctx.ValidKeywords, "then") {
		t.Errorf("expected 'then' in ValidKeywords, got %v", ctx.ValidKeywords)
	}
	// Assignment operators and comma should not be valid in if condition
	for _, op := range ctx.ValidOperators {
		if op.Op == "," || op.Op == "=" || op.Op == "|=" || op.Op == "+=" || op.Op == "-=" || op.Op == "*=" || op.Op == "/=" || op.Op == "%=" || op.Op == "//=" {
			t.Errorf("operator %q should not be valid in if condition", op.Op)
		}
	}
}

func TestCompletionIfThenBodyContext(t *testing.T) {
	ctx := ParseForCompletion("if . > 0 then . ")
	if ctx.If == nil || ctx.If.Section != "then" {
		t.Errorf("expected If context with section 'then', got %v", ctx.If)
	}
	assertHasExpected(t, ctx, ExpectedKeyword)
	if !slices.Contains(ctx.ValidKeywords, "elif") || !slices.Contains(ctx.ValidKeywords, "else") || !slices.Contains(ctx.ValidKeywords, "end") {
		t.Errorf("expected 'elif', 'else', 'end' in ValidKeywords, got %v", ctx.ValidKeywords)
	}
}

func TestCompletionIfElseBodyContext(t *testing.T) {
	ctx := ParseForCompletion("if . > 0 then . else . ")
	if ctx.If == nil || ctx.If.Section != "else" {
		t.Errorf("expected If context with section 'else', got %v", ctx.If)
	}
	assertHasExpected(t, ctx, ExpectedKeyword)
	if !slices.Contains(ctx.ValidKeywords, "end") {
		t.Errorf("expected 'end' in ValidKeywords, got %v", ctx.ValidKeywords)
	}
}

func TestCompletionAfterTry(t *testing.T) {
	ctx := ParseForCompletion("try ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterTryBody(t *testing.T) {
	ctx := ParseForCompletion("try .a ")
	assertHasExpected(t, ctx, ExpectedKeyword)
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionInReduce(t *testing.T) {
	ctx := ParseForCompletion("reduce .[] as $item (")
	if ctx.Reduce == nil {
		t.Fatal("expected Reduce context")
	}
	if ctx.Reduce.Section != "init" {
		t.Errorf("expected section 'init', got %q", ctx.Reduce.Section)
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionInReduceAfterSemicolon(t *testing.T) {
	ctx := ParseForCompletion("reduce .[] as $item (0; ")
	if ctx.Reduce == nil {
		t.Fatal("expected Reduce context")
	}
	if ctx.Reduce.Section != "update" {
		t.Errorf("expected section 'update', got %q", ctx.Reduce.Section)
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionInForeach(t *testing.T) {
	ctx := ParseForCompletion("foreach .[] as $item (0; ")
	if ctx.Reduce == nil {
		t.Fatal("expected Reduce context")
	}
	if !ctx.Reduce.IsForeach {
		t.Error("expected IsForeach to be true")
	}
	if ctx.Reduce.Section != "update" {
		t.Errorf("expected section 'update', got %q", ctx.Reduce.Section)
	}
}

func TestCompletionAsBinding(t *testing.T) {
	ctx := ParseForCompletion(".foo as ")
	if ctx.InAsPattern {
		t.Error("InAsPattern should not be set yet")
	}
}

func TestCompletionAsBindingPattern(t *testing.T) {
	ctx := ParseForCompletion(".foo as $")
	// After $, we're in a pattern context
	if ctx.Function != nil {
		t.Fatal("did not expect Function context")
	}
}

func TestCompletionPartialString(t *testing.T) {
	ctx := ParseForCompletion(`"fo`)
	if ctx.PartialString != "fo" {
		t.Errorf("expected PartialString 'fo', got %q", ctx.PartialString)
	}
	if ctx.StringQuote != '"' {
		t.Errorf("expected StringQuote \", got %c", ctx.StringQuote)
	}
	assertHasExpected(t, ctx, ExpectedStringClose)
}

func TestCompletionInStringInterp(t *testing.T) {
	ctx := ParseForCompletion(`"hello \(`)
	if !ctx.InStringInterp {
		t.Error("expected InStringInterp to be true")
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionFormatName(t *testing.T) {
	ctx := ParseForCompletion("@")
	if !ctx.InFormat {
		t.Error("expected InFormat to be true")
	}
	assertHasExpected(t, ctx, ExpectedFormatName)
}

func TestCompletionAfterFormat(t *testing.T) {
	ctx := ParseForCompletion("@base64 ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionInParen(t *testing.T) {
	ctx := ParseForCompletion("(.")
	assertHasExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionAfterComparison(t *testing.T) {
	ctx := ParseForCompletion(".a > ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterAlternative(t *testing.T) {
	ctx := ParseForCompletion(".foo // ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterAnd(t *testing.T) {
	ctx := ParseForCompletion(".a and ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionInDef(t *testing.T) {
	ctx := ParseForCompletion("def ")
	if !ctx.InDef {
		t.Error("expected InDef to be true")
	}
}

func TestCompletionAfterDefName(t *testing.T) {
	ctx := ParseForCompletion("def myfunc")
	assertHasExpected(t, ctx, ExpectedDefColon)
	if ctx.DefName != "myfunc" {
		t.Errorf("expected DefName 'myfunc', got %q", ctx.DefName)
	}
}

func TestCompletionAfterDefBody(t *testing.T) {
	ctx := ParseForCompletion("def myfunc: . + 1; ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterLabel(t *testing.T) {
	ctx := ParseForCompletion("label $out | ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterBreak(t *testing.T) {
	ctx := ParseForCompletion("break ")
	assertHasExpected(t, ctx, ExpectedDollar)
}

func TestCompletionAfterSliceStart(t *testing.T) {
	ctx := ParseForCompletion(".[2:")
	// After slice start, we can have end expression or ]
	assertHasExpected(t, ctx, ExpectedClosingBracket)
}

func TestCompletionInSlice(t *testing.T) {
	ctx := ParseForCompletion(".[2:")
	assertNotExpected(t, ctx, ExpectedExpression)
}

func TestCompletionAfterNegate(t *testing.T) {
	ctx := ParseForCompletion("-")
	assertHasExpected(t, ctx, ExpectedExpression)
}

// --- Bug fix tests ---

func TestCompletionAfterDotNoAfterDotForBracket(t *testing.T) {
	ctx := ParseForCompletion(".[2:]")
	if ctx.AfterDot {
		t.Error("AfterDot should be false after bracket access")
	}
}

func TestCompletionAfterDotNoAfterDotForIndex(t *testing.T) {
	ctx := ParseForCompletion(".[0]")
	if ctx.AfterDot {
		t.Error("AfterDot should be false after index access")
	}
}

func TestCompletionAfterDotNoAfterDotForQuotedField(t *testing.T) {
	ctx := ParseForCompletion(`."foo"`)
	if ctx.AfterDot {
		t.Error("AfterDot should be false after quoted field access")
	}
}

func TestCompletionPostfixBracketNoAfterDot(t *testing.T) {
	ctx := ParseForCompletion(".foo[0]")
	if ctx.AfterDot {
		t.Error("AfterDot should be false after .foo[0]")
	}
}

func TestCompletionPostfixQuotedFieldNoAfterDot(t *testing.T) {
	ctx := ParseForCompletion(`.foo."bar"`)
	if ctx.AfterDot {
		t.Error("AfterDot should be false after .foo.\"bar\"")
	}
}

func TestCompletionIdentifierNoSpuriousClosingParen(t *testing.T) {
	ctx := ParseForCompletion("foo")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertNotExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionIdentifierInFunctionGetsClosingParen(t *testing.T) {
	ctx := ParseForCompletion("map(foo")
	assertHasExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionIdentifierInParenGetsClosingParen(t *testing.T) {
	ctx := ParseForCompletion("(foo")
	assertHasExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionIfConditionWithPipe(t *testing.T) {
	ctx := ParseForCompletion("if .a | .b then ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionIfCompleteWithPipe(t *testing.T) {
	ctx := ParseForCompletion("if .a | .b then . else . end ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionReduceSourceWithPipe(t *testing.T) {
	ctx := ParseForCompletion("reduce .a | .b as $x (")
	if ctx.Reduce == nil {
		t.Fatal("expected Reduce context")
	}
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionForeachSourceWithPipe(t *testing.T) {
	ctx := ParseForCompletion("foreach .a | .b as $x (")
	if ctx.Reduce == nil {
		t.Fatal("expected Reduce context")
	}
	if !ctx.Reduce.IsForeach {
		t.Error("expected IsForeach to be true")
	}
}

func TestCompletionFunctionArgWithPipe(t *testing.T) {
	ctx := ParseForCompletion("map(.a | .b) ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionArrayElementWithPipe(t *testing.T) {
	ctx := ParseForCompletion("[.a | .b] ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionObjectValueWithPipe(t *testing.T) {
	ctx := ParseForCompletion("{foo: .a | .b} ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionBracketAccessWithPipe(t *testing.T) {
	ctx := ParseForCompletion(".[.a | .b] ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionStringInterpWithPipe(t *testing.T) {
	ctx := ParseForCompletion(`"hello \(.a | .b)"`)
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionReduceExpectsOpeningParen(t *testing.T) {
	ctx := ParseForCompletion("reduce .[] as $x ")
	assertHasExpected(t, ctx, ExpectedOpeningParen)
	assertNotExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionForeachExpectsOpeningParen(t *testing.T) {
	ctx := ParseForCompletion("foreach .[] as $x ")
	assertHasExpected(t, ctx, ExpectedOpeningParen)
	assertNotExpected(t, ctx, ExpectedClosingParen)
}

func TestCompletionNestedDefRestoresDefName(t *testing.T) {
	ctx := ParseForCompletion("def f: def g: .; g; f")
	// After the inner def's semicolon and "g;", we're parsing the rest "f"
	// DefName should NOT be "g" (the inner def's name)
	if ctx.InDef {
		t.Error("InDef should be false after nested def completes")
	}
	if ctx.DefName == "g" {
		t.Error("DefName should not be 'g' after inner def completes")
	}
}

func TestCompletionNestedDefRestoresInDef(t *testing.T) {
	ctx := ParseForCompletion("def f: def g: .; g; ")
	// After inner def, we're parsing the rest of outer def
	// InDef should be false (outer def's rest is the top-level body after ;)
	if ctx.DefName == "g" {
		t.Error("DefName should not be 'g' after inner def completes")
	}
}

func TestCompletionPatternAlternativeAtCursor(t *testing.T) {
	ctx := ParseForCompletion(".foo as [$a] ?// ")
	assertHasExpected(t, ctx, ExpectedDollar)
}

func TestCompletionPatternAlternativeComplete(t *testing.T) {
	ctx := ParseForCompletion(".foo as [$a] ?// $b | ")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionIfThenElseEndWithPipes(t *testing.T) {
	ctx := ParseForCompletion("if .a | .b then .c | .d else .e | .f end ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionElifWithPipe(t *testing.T) {
	ctx := ParseForCompletion("if .a then .b elif .c | .d then .e end ")
	assertHasExpected(t, ctx, ExpectedOperator)
}

func TestCompletionDotFieldThenBracketNoExpression(t *testing.T) {
	// .foo.[0] — the . before [ is an error in jq, but the completion parser
	// should not add ExpectedExpression (the [ will be handled by postfix)
	ctx := ParseForCompletion(".foo.[0]")
	assertNotExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedOperator)
}

// --- Edge case tests for completion parser fixes ---

func TestCompletionBreakDollar(t *testing.T) {
	// After "break $", we need a label name, not operators
	ctx := ParseForCompletion("break $")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertNotExpected(t, ctx, ExpectedOperator)
}

func TestCompletionLabelDollar(t *testing.T) {
	// After "label $", we need a label name
	ctx := ParseForCompletion("label $")
	assertHasExpected(t, ctx, ExpectedExpression)
}

func TestCompletionObjectAfterComma(t *testing.T) {
	// After comma in object, we need a new key expression
	ctx := ParseForCompletion("{a: 1, ")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingBrace)
	assertNotExpected(t, ctx, ExpectedOperator)
	if ctx.Object == nil {
		t.Fatal("expected Object context")
	}
	if !ctx.Object.InKey {
		t.Error("expected InKey to be true after comma in object")
	}
}

func TestCompletionObjectPartialKeyNoOperators(t *testing.T) {
	// After "b" as partial key, operators should not be suggested
	ctx := ParseForCompletion("{a: 1, b")
	assertNotExpected(t, ctx, ExpectedOperator)
	assertHasExpected(t, ctx, ExpectedColon)
}

func TestCompletionPatternAlternativePartial(t *testing.T) {
	// After ? in a pattern context, it's the start of ?//
	// Should not report Expression (it's not an expression context)
	ctx := ParseForCompletion(". as {$a} ?")
	assertNotExpected(t, ctx, ExpectedExpression)
}

func TestCompletionFunctionArgsSemicolonSeparator(t *testing.T) {
	// After semicolon in function args, we expect a new expression
	ctx := ParseForCompletion("limit(3; ")
	assertHasExpected(t, ctx, ExpectedExpression)
	assertHasExpected(t, ctx, ExpectedClosingParen)
	if ctx.Function == nil {
		t.Fatal("expected Function context")
	}
	if ctx.Function.ArgIndex != 1 {
		t.Errorf("expected arg index 1, got %d", ctx.Function.ArgIndex)
	}
}
