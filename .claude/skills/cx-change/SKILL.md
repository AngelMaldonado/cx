---
name: cx-change
description: Create and manage structured changes tracked in docs/changes/. Activate when the developer wants to start a new feature, reference an existing change, or update change status.
---

# Skill: cx-change

## Description
Create and manage structured changes. A change is a set of related modifications tracked in `docs/changes/` with proposal, design, and task documents.

## Triggers
- Developer wants to start a new feature or change
- Developer references an existing change by name
- Developer wants to update change status

## Steps
1. Run `cx change <name>` to create a new change
2. Fill in proposal.md with the problem statement and proposed solution
3. Fill in design.md with technical approach
4. Track tasks in tasks.md with Linear issue references

## Rules
- Change names must be kebab-case
- Every change must have a proposal before design work begins
- Link Linear issues in tasks.md for tracking
