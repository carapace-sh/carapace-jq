package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-jq/pkg/actions/tools/jq"
	jqparser "github.com/carapace-sh/carapace-jq/pkg/jq"
	"github.com/spf13/cobra"
)

var filterCmd = &cobra.Command{
	Use:   "filter <expression>",
	Short: "Parse a jq filter expression",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		expression, err := jqparser.Parse(args[0])
		if err != nil {
			return err
		}
		m, err := json.MarshalIndent(expression, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(m))
		return nil
	},
}

var filterCompleteCmd = &cobra.Command{
	Use:   "filter-complete <expression>",
	Short: "Get completion context for a jq filter expression",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := jqparser.ParseForCompletion(args[0])
		m, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(m))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(filterCmd)
	rootCmd.AddCommand(filterCompleteCmd)

	// jq filter expressions can contain spaces, so the shell may split them
	// into multiple words. Use PositionalAnyCompletion to rejoin all args
	// (except the last toComplete) into a single expression string.
	//
	// Carapace applies prefix filtering against c.Value after the action
	// returns. When the expression is split across multiple args (shell word
	// boundaries), c.Value is the partial last word and filtering works. But
	// when the entire expression is in c.Value (e.g. "if .a "), completions
	// like "then" or "|" get filtered out since they don't start with that
	// prefix. We fix this by using Prefix() to strip the already-typed part
	// from c.Value before the action runs, so prefix filtering operates on
	// just the last token.
	filterCompletion := carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		expr := strings.TrimLeft(strings.Join(c.Args, " ")+" "+c.Value, " ")

		// Compute the already-typed prefix: everything before the last
		// space-separated token in the combined expression.
		typedPrefix := ""
		partialToken := c.Value
		if lastSpace := strings.LastIndex(c.Value, " "); lastSpace >= 0 {
			typedPrefix = c.Value[:lastSpace+1]
			partialToken = c.Value[lastSpace+1:]
		}

		// When there are no spaces in c.Value and the parser indicates the
		// expression is complete (ExpectedOperator, not ExpectedExpression),
		// the current word is a complete jq expression. Set partialToken to
		// empty so prefix filtering doesn't remove operator/keyword completions.
		if !strings.Contains(c.Value, " ") {
			ctx := jqparser.ParseForCompletion(expr)
			if !hasExpectedExpr(ctx) {
				typedPrefix = expr
				partialToken = ""
			}
		}

		c.Value = partialToken
		return jq.ActionFilterComplete(expr).Invoke(c).Prefix(typedPrefix).ToA()
	})

	carapace.Gen(filterCmd).PositionalAnyCompletion(filterCompletion)
	carapace.Gen(filterCompleteCmd).PositionalAnyCompletion(filterCompletion)
}

func hasExpectedExpr(ctx *jqparser.CompletionContext) bool {
	for _, t := range ctx.ExpectedTokens {
		if t == jqparser.ExpectedExpression {
			return true
		}
	}
	return false
}
