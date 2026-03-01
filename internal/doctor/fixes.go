package doctor

type FixableItem struct {
	Index int
	Label string
	Fix   func() error
}

func CollectFixable(groups []CheckGroup) []FixableItem {
	var items []FixableItem
	idx := 1
	for _, g := range groups {
		for _, r := range g.Results {
			if r.Fixable && r.FixFunc != nil {
				items = append(items, FixableItem{
					Index: idx,
					Label: r.FixLabel,
					Fix:   r.FixFunc,
				})
				idx++
			}
		}
	}
	return items
}

func ApplyFixes(items []FixableItem) []error {
	var errs []error
	for _, item := range items {
		if err := item.Fix(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
