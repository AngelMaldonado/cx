# Skill: cx-review

## Description
Post-implementation review skill for the Reviewer agent. Spawned by the Master after implementation work completes, the Reviewer reads `git diff`, compares changes against the change's spec/plan files, evaluates code quality, and verifies test coverage. It acts as a quality gate before work is presented to the developer as done.

## Triggers
- Master receives completion report from a Supervisor or Contractor
- Implementation work is done and ready for review
- Developer explicitly asks to review changes ("review my changes", "check this against the spec")

## Steps
1. Run `git diff` (or `git diff --staged` if changes are staged) to get the full set of changes made.
2. Identify which change this work belongs to. Load the change's artifacts:
   - Read `docs/changes/<name>/proposal.md` — understand the intent
   - Read `docs/changes/<name>/design.md` — understand the planned approach
   - Read any delta specs in `docs/changes/<name>/specs/` — understand spec modifications
3. Load the relevant canonical specs using `cx context --load spec <area>` for each affected spec area.
4. **Spec alignment check**: Compare the code changes against the proposal, design, and specs:
   - Does the implementation match what was proposed?
   - Are there changes that go beyond the stated scope?
   - Are there parts of the proposal that were not implemented?
   - Do delta specs accurately reflect the actual changes to system behavior?
5. **Code quality check**: Review the code changes for:
   - Consistency with existing patterns and conventions in the codebase
   - Obvious issues (error handling gaps, missing edge cases, naming inconsistencies)
   - Security concerns (input validation, injection risks, auth checks)
   - Performance concerns (N+1 queries, unnecessary allocations, blocking calls)
6. **Test coverage check**:
   - Do new functions/endpoints have corresponding tests?
   - Were existing tests updated for modified behavior?
   - Run tests if possible and report results
7. Produce a structured review report:

```
## Review Summary

**Status**: PASS | NEEDS_FIXES | NEEDS_DISCUSSION

### Spec Alignment
- [PASS|ISSUE] <alignment assessment>
- Scope: <in scope / out of scope changes noted>

### Code Quality
- [PASS|ISSUE] <quality assessment per file>
- Patterns: <follows conventions / deviates in X way>

### Test Coverage
- [PASS|ISSUE] <coverage assessment>
- Missing: <untested areas>
- Results: <test run output if available>

### Recommendations
- <specific fix or improvement, if any>
```

8. Return the review report to the Master. Include:
   - Overall pass/fail recommendation
   - List of specific issues with file:line references
   - Observations worth saving (via cx-memory) if any non-obvious patterns or constraints were discovered

## Rules
- Never modify code or files — the Reviewer is strictly read-only
- Never interact with the developer directly — return the review to the Master
- Be specific: every issue must reference a file and line number
- Distinguish between blocking issues (NEEDS_FIXES) and discussion points (NEEDS_DISCUSSION)
- A PASS does not mean perfect code — it means the implementation aligns with the spec, follows conventions, and has adequate test coverage
- Do not suggest alternative implementations — only identify issues with the current one
- If the review reveals that the spec itself is incomplete or ambiguous, flag this as NEEDS_DISCUSSION rather than failing the review
- If no change artifacts exist (direct dispatch without a formal change), review against the original task description from the Master
