package brainstorm

import "fmt"

func MasterfileTemplate(name string) string {
	return fmt.Sprintf(`# Masterfile: %s

## Problem
<what pain point or opportunity is being addressed>

## Context
<relevant background — what exists today, what constraints are known>

## Direction
<the emerging solution direction — updated as the brainstorm evolves>

## Open Questions
- <question 1>
- <question 2>

## Files to Modify
<specific files and what changes in each>

## Risks
<what could go wrong and how to mitigate>

## Testing
<how to verify the implementation>

## References
<links to specs, external docs, prior art>
`, name)
}
