---
name: cx-review
description: Review code changes, pull requests, and documents for quality, correctness, and adherence to project conventions. Activate when the developer asks for a review or a PR is opened.
---

# Skill: cx-review

## Description
Review code and verify planning artifacts. Two modes: **code review** (post-implementation quality gate) and **artifact verification** (cross-check planning docs against current code and specs).

## Triggers
- Developer asks for a code review
- Pull request is opened or updated
- Planning artifact generated (masterfile, proposal, design, tasks) — triggers artifact verification
- Document review is requested

## Steps

### Code review mode
1. Run `cx memory search --change <name>` to load change-scoped observations and decisions
2. Read the changes in full context
3. Check against project conventions and `.cx/cx.yaml` rules
4. Identify issues: bugs, style, performance, security
5. Provide specific, constructive feedback

### Artifact verification mode
1. Read the planning artifact in full
2. Read `docs/specs/` for current canonical specs
3. Explore the actual code referenced or implied by the artifact
4. Cross-check every factual claim: file paths, function names, architectural assumptions, existing behavior descriptions
5. Return a structured report: **Verified** (matches reality), **Inaccurate** (contradicts reality, with evidence), **Unverifiable** (doesn't exist yet — fine for planning), **Missing context** (important things the artifact should reference)
6. Do NOT judge the quality of the plan — only verify factual accuracy

## Rules
- Be specific — reference line numbers and files
- Distinguish between blocking issues and suggestions
- Check for consistency with existing code patterns
- Never approve changes you haven't fully reviewed
- Reviewer is read-only — never writes memory via `cx memory save`
- Significant recurring patterns should be returned to Master in the review report; Master decides whether to save as observations
