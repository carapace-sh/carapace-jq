package jq

import (
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/sandbox"
)

func TestActionFormatsValues(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFormats()
	})(func(s *sandbox.Sandbox) {
		s.Run("").ExpectNot(carapace.ActionValues(
			"@@text", "@@json", "@@html", "@@uri",
			"@@csv", "@@tsv", "@@sh", "@@base64", "@@base64d",
		))
	})
}

func TestActionFiltersExpressionContextNoDoubleAt(t *testing.T) {
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
