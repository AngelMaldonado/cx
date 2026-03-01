package doctor

import (
	"fmt"

	"github.com/amald/cx/internal/ui"
)

func PrintReport(groups []CheckGroup) (errors, warnings int) {
	for _, g := range groups {
		ui.PrintHeader(g.Name)
		for _, r := range g.Results {
			switch r.Severity {
			case Pass:
				ui.PrintSuccess(r.Message)
			case Warning:
				ui.PrintWarning(r.Message)
				warnings++
			case Error:
				ui.PrintError(r.Message)
				errors++
			}
		}
	}
	return errors, warnings
}

func PrintFixableList(items []FixableItem) {
	ui.PrintHeader("fixable issues")
	for _, item := range items {
		fmt.Printf("    %s %s\n",
			ui.StyleAccent.Render(fmt.Sprintf("%d.", item.Index)),
			ui.StyleItem.Render(item.Label),
		)
	}
}
