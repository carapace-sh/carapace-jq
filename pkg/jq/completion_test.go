package jq

import (
	"testing"
)

func assertHasExpected(t *testing.T, ctx *CompletionContext, tok ExpectedToken) {
	t.Helper()
	for _, e := range ctx.ExpectedTokens {
		if e == tok {
			return
		}
	}
	t.Errorf("expected token %v not found in %v", tok, ctx.ExpectedTokens)
}

func assertNotExpected(t *testing.T, ctx *CompletionContext, tok ExpectedToken) {
	t.Helper()
	for _, e := range ctx.ExpectedTokens {
		if e == tok {
			t.Errorf("token %v should not be in %v", tok, ctx.ExpectedTokens)
			return
		}
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

func assertNoOperator(t *testing.T, ctx *CompletionContext, op string) {
	t.Helper()
	for _, o := range ctx.ValidOperators {
		if o.Op == op {
			t.Errorf("operator %q should not be in %v", op, ctx.ValidOperators)
			return
		}
	}
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
	ctx := ParseForCompletion("limit(3, ")
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
