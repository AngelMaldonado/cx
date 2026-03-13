package change

import "fmt"

func ProposalTemplate(name string) string {
	return fmt.Sprintf(`# Proposal: %s

## Problem
<what's wrong or missing — the pain that motivates this change>

## Approach
<high-level solution direction — what will be done, not how>

## Scope
<what's in scope and what's explicitly out of scope>

## Affected Specs
<which spec areas will have deltas — e.g., gas-monitoring, device-communication>
`, name)
}

func DesignTemplate(name string) string {
	return fmt.Sprintf(`# Design: %s

## Architecture
<how the solution fits into the existing system — components, data flow, interfaces>

## Technical Decisions
<key choices made during design — libraries, patterns, approaches>

## Implementation Notes
<anything the implementing agent needs to know — gotchas, constraints, dependencies>
`, name)
}

func TasksTemplate(name string) string {
	return fmt.Sprintf(`# Tasks: %s

## Linear Issues
- PROJ-100: <task description>
- PROJ-101: <task description>
- PROJ-102: <task description>

## Implementation Notes
<technical notes relevant to implementation — ordering, dependencies between tasks>
`, name)
}
