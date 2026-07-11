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
	filterCompletion := carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		expr := strings.Join(c.Args, " ") + " " + c.Value
		return jq.ActionFilterComplete(expr)
	})

	carapace.Gen(filterCmd).PositionalAnyCompletion(filterCompletion)
	carapace.Gen(filterCompleteCmd).PositionalAnyCompletion(filterCompletion)
}
