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

		// Operators, closing tokens, and keywords can coexist (e.g. after a
		// complete expression inside if/try, both operators and elif/else/end
		// are valid). Batch them together instead of returning exclusively.
		batch := carapace.Batch()

		if hasExpected(ctx, jqparser.ExpectedOperator) && len(ctx.ValidOperators) > 0 {
			ops := make([]jqValidOperator, 0, len(ctx.ValidOperators))
			for _, op := range ctx.ValidOperators {
				ops = append(ops, jqValidOperator{Op: op.Op, Description: op.Description})
			}
			batch = append(batch, ActionOperators(ops).NoSpace())
		}

		if hasExpected(ctx, jqparser.ExpectedClosingParen) {
			batch = append(batch, carapace.ActionValues(")"))
		}
		if hasExpected(ctx, jqparser.ExpectedClosingBracket) {
			batch = append(batch, carapace.ActionValues("]"))
		}
	if hasExpected(ctx, jqparser.ExpectedClosingBrace) {
			batch = append(batch, carapace.ActionValues("}"))
		}
		if hasExpected(ctx, jqparser.ExpectedOpeningParen) {
			batch = append(batch, carapace.ActionValues("(").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedComma) {
			batch = append(batch, carapace.ActionValues(","))
		}
		if hasExpected(ctx, jqparser.ExpectedPipe) {
			batch = append(batch, carapace.ActionValues("|").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedColon) {
			batch = append(batch, carapace.ActionValues(":").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedDefColon) {
			batch = append(batch, carapace.ActionValues(":").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedSemicolon) {
			batch = append(batch, carapace.ActionValues(";").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedDefSemicolon) {
			batch = append(batch, carapace.ActionValues(";").NoSpace())
		}
		if hasExpected(ctx, jqparser.ExpectedKeyword) {
			batch = append(batch, actionForKeywordTokens(ctx))
		}

		if len(batch) > 0 {
			return batch.ToA()
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

// actionForKeywordTokens returns keyword token completions filtered by ValidKeywords.
// When ValidKeywords is populated, only those keywords are offered. Otherwise,
// all keyword tokens are offered as a fallback.
func actionForKeywordTokens(ctx *jqparser.CompletionContext) carapace.Action {
	if len(ctx.ValidKeywords) > 0 {
		descs := map[string]string{
			"then":  "Then branch of if expression",
			"elif":  "Else-if branch",
			"else":  "Else branch",
			"end":   "End of if expression",
			"catch": "Error handler for try",
			"as":    "Variable binding",
		}
		vals := make([]string, 0, len(ctx.ValidKeywords))
		described := make([]string, 0, len(ctx.ValidKeywords)*2)
		for _, kw := range ctx.ValidKeywords {
			vals = append(vals, kw)
			described = append(described, kw, descs[kw])
		}
		return carapace.ActionValuesDescribed(described...).UidF(Uid("keyword-token")).NoSpace()
	}
	return ActionKeywordTokens()
}
