---
name: cx-scout
description: Explore and map unfamiliar codebases. Activate when the developer asks to understand a new codebase, is onboarding, or exploring an unfamiliar area of the code.
---

# Skill: cx-scout

## Description
Explore and map unfamiliar codebases. Builds understanding of project structure, key patterns, and important files.

## Triggers
- Developer asks to understand a new codebase
- Onboarding to a project
- Exploring an unfamiliar area of the code

## Steps
1. Map the top-level directory structure
2. Identify key entry points and configuration
3. Trace important code paths
4. Document findings as observations

## Rules
- Start broad, then go deep on areas of interest
- Document findings as you go
- Don't make changes while scouting — observe only
- Scout is read-only — do NOT call `cx memory save` directly
- Return all discoveries to the Master in the summary; Master decides whether to save as observations via `cx memory save --type observation`
