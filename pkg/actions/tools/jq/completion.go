package jq

import (
	"github.com/carapace-sh/carapace"
	jqparser "github.com/carapace-sh/carapace-jq/pkg/jq"
)

// ActionFilterComplete generates carapace completions for a jq filter expression
// based on the completion context derived from the partial input.
func ActionFilterComplete(input string) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		ctx := jqparser.ParseForCompletion(input)

		// Inside a string literal — offer string close or interpolated expression
		if ctx.StringQuote != 0 && !ctx.InStringInterp {
			return carapace.ActionValues("\"").NoSpace()
		}

		// Inside string interpolation \(...)
		if ctx.InStringInterp {
			return actionForExpectedExpression(ctx)
		}

		// Inside a format name @...
		if ctx.InFormat {
			return ActionFormatStrings()
		}

		// Inside an as pattern — offer $ prefix
		if ctx.InAsPattern {
			return carapace.ActionValues("$").NoSpace()
		}

		// Inside a def — offer function name completion is context-dependent
		if ctx.InDef && ctx.DefName == "" {
			return carapace.ActionMessage("function name")
		}

		// After break — expect $
		if hasExpected(ctx, jqparser.ExpectedDollar) {
			return carapace.ActionValues("$").NoSpace()
		}

		// Inside reduce/foreach
		if ctx.Reduce != nil {
			return actionForExpectedExpression(ctx)
		}

		// Inside object construction
		if ctx.Object != nil {
			if ctx.Object.InKey {
				return carapace.ActionMessage("object key")
			}
			if ctx.Object.InValue {
				return actionForExpectedExpression(ctx)
			}
		}

		// After dot — field name completion
		if ctx.AfterDot {
			return carapace.ActionMessage("field name")
		}

		// Inside a function call
		if ctx.Function != nil {
			return actionForExpectedExpression(ctx)
		}

		// Check expected tokens
		if hasExpected(ctx, jqparser.ExpectedExpression) {
			return actionForExpectedExpression(ctx)
		}

		// Operators
		if hasExpected(ctx, jqparser.ExpectedOperator) && len(ctx.ValidOperators) > 0 {
			ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
			for _, op := range ctx.ValidOperators {
				ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
			}
			return ActionOperators(ops).NoSpace()
		}

		// Closing tokens
		if hasExpected(ctx, jqparser.ExpectedClosingParen) {
			return carapace.ActionValues(")")
		}
		if hasExpected(ctx, jqparser.ExpectedClosingBracket) {
			return carapace.ActionValues("]")
		}
		if hasExpected(ctx, jqparser.ExpectedClosingBrace) {
			return carapace.ActionValues("}")
		}
		if hasExpected(ctx, jqparser.ExpectedOpeningParen) {
			return carapace.ActionValues("(").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedComma) {
			return carapace.ActionValues(",")
		}
		if hasExpected(ctx, jqparser.ExpectedPipe) {
			return carapace.ActionValues("|").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedColon) {
			return carapace.ActionValues(":").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedDefColon) {
			return carapace.ActionValues(":").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedSemicolon) {
			return carapace.ActionValues(";").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedDefSemicolon) {
			return carapace.ActionValues(";").NoSpace()
		}
		if hasExpected(ctx, jqparser.ExpectedKeyword) {
			return ActionKeywordTokens()
		}

		return carapace.ActionValues()
	})
}

func actionForExpectedExpression(ctx *jqparser.CompletionContext) carapace.Action {
	batch := carapace.Batch()

	// Builtins (function names)
	batch = append(batch, ActionBuiltins())

	// Keywords (if, try, reduce, foreach, def, label, break)
	batch = append(batch, ActionKeywords())

	// Literals (true, false, null)
	batch = append(batch, ActionLiterals())

	// Special filters (. and ..)
	batch = append(batch, ActionSpecialFilters())

	// Format strings (@base64, @uri, etc.)
	batch = append(batch, ActionFormatStrings().Prefix("@"))

	// If we have a partial identifier, filter by it
	if ctx.PartialIdent != "" {
		// carapace handles prefix filtering automatically
	}

	return batch.ToA()
}

func hasExpected(ctx *jqparser.CompletionContext, tok jqparser.ExpectedToken) bool {
	for _, e := range ctx.ExpectedTokens {
		if e == tok {
			return true
		}
	}
	return false
}
