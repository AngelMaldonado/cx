---
name: cx-review
description: Review code changes, pull requests, and documents for quality, correctness, and adherence to project conventions. Activate when the developer asks for a review or a PR is opened.
---

# Skill: cx-review

## Description
Review code changes, pull requests, and documents for quality, correctness, and adherence to project conventions.

## Triggers
- Developer asks for a code review
- Pull request is opened or updated
- Document review is requested

## Steps

### 1. Load change context
- Run `cx memory search --change <name>` to load change-scoped observations and decisions
- These inform the review: prior constraints, design decisions made during implementation

### 2. Read the changes in context
### 3. Check against project conventions and .cx/cx.yaml rules
### 4. Identify issues: bugs, style, performance, security
### 5. Provide specific, constructive feedback

## Rules
- Be specific — reference line numbers and files
- Distinguish between blocking issues and suggestions
- Check for consistency with existing code patterns
- Never approve changes you haven't fully reviewed
- Reviewer is read-only — never writes memory via cx memory save
- Significant recurring patterns should be returned to Master in the review report; Master decides whether to save as observations
