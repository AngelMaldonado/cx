---
name: cx-contract
description: Manage API and interface contracts. Activate when the developer defines or modifies an API endpoint, changes an interface boundary, or requests contract validation.
---

# Skill: cx-contract

## Description
Manage API and interface contracts. Tracks contract definitions, validates changes for backward compatibility, and generates documentation.

## Triggers
- Developer defines or modifies an API endpoint
- Developer changes an interface boundary
- Contract validation is requested

## Steps
1. Identify the contract being modified
2. Check for backward compatibility with existing consumers
3. Document the change in the appropriate spec
4. Update generated documentation if applicable

## Rules
- Breaking changes require explicit developer approval
- All contracts must be documented in specs
- Version contracts when making breaking changes
