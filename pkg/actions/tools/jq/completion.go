package jq

import (
	"slices"
	"strings"

	"github.com/carapace-sh/carapace"
	jqparser "github.com/carapace-sh/carapace-jq/pkg/jq"
)

func ActionFilters() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		expr := c.Value
		ctx := jqparser.ParseForCompletion(expr)

		typedPrefix := ""
		partialToken := expr
		if lastSpace := strings.LastIndex(expr, " "); lastSpace >= 0 {
			typedPrefix = expr[:lastSpace+1]
			partialToken = expr[lastSpace+1:]
		}

		if !strings.Contains(expr, " ") && !hasExpected(ctx, jqparser.ExpectedExpression) {
			typedPrefix = expr
			partialToken = ""
		}

		c.Value = partialToken
		return actionForCompletionContext(ctx).Invoke(c).Prefix(typedPrefix).ToA()
	})
}

func actionForCompletionContext(ctx *jqparser.CompletionContext) carapace.Action {
	if ctx.StringQuote != 0 && !ctx.InStringInterp {
		return carapace.ActionValues("\"").NoSpace()
	}

	if ctx.InStringInterp {
		return actionForExpectedExpression(ctx)
	}

	if ctx.InFormat {
		return ActionFormatStrings()
	}

	if ctx.InAsPattern {
		return carapace.ActionValues("$").NoSpace()
	}

	if ctx.InDef && ctx.DefName == "" {
		return carapace.ActionMessage("function name")
	}

	if hasExpected(ctx, jqparser.ExpectedDollar) {
		return carapace.ActionValues("$").NoSpace()
	}

	if ctx.Reduce != nil {
		return actionForExpectedExpression(ctx)
	}

	if ctx.Object != nil {
		if ctx.Object.InKey {
			return carapace.ActionMessage("object key")
		}
		if ctx.Object.InValue {
			return actionForExpectedExpression(ctx)
		}
	}

	if ctx.AfterDot {
		return carapace.ActionMessage("field name")
	}

	if ctx.Function != nil {
		return actionForExpectedExpression(ctx)
	}

	if hasExpected(ctx, jqparser.ExpectedExpression) {
		return actionForExpectedExpression(ctx)
	}

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
}

func actionForExpectedExpression(ctx *jqparser.CompletionContext) carapace.Action {
	batch := carapace.Batch()

	batch = append(batch, ActionBuiltins())
	batch = append(batch, ActionKeywords())
	batch = append(batch, ActionLiterals())
	batch = append(batch, ActionSpecialFilters())
	batch = append(batch, ActionFormatStrings().Prefix("@"))

	return batch.ToA()
}

func hasExpected(ctx *jqparser.CompletionContext, tok jqparser.ExpectedToken) bool {
	return slices.Contains(ctx.ExpectedTokens, tok)
}

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
