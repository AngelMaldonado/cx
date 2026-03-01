# Skill: cx-doctor

## Description
Teaches the Master how to run project health checks and fix common issues. `cx doctor` validates the docs/ structure, memory files, git hooks, and MCP configuration. With `--fix`, it auto-repairs what it can.

## Triggers
- Developer asks about project health ("is everything set up right?", "check the project", "run doctor")
- Agent encounters an error from another cx command (suggests running doctor to diagnose)
- After `cx init` to verify the setup
- Periodically during long work sessions (good practice to check for drift)

## Steps
1. Run: `cx doctor`
2. Read the output. It is organized into four sections:
   - **docs/ structure**: verifies required files and directories exist
   - **memory health**: validates frontmatter, required sections, cross-references
   - **git hooks**: checks post-merge and post-checkout hooks are installed
   - **MCP config**: validates required MCP servers are configured
3. Each check shows one of: pass, warning, or error with a description.
4. Present the results to the developer in a clear summary. Focus on errors first, then warnings.
5. For fixable issues, suggest running `cx doctor --fix` which auto-repairs:
   - Missing directories (creates them)
   - Missing or outdated git hooks (installs them)
   - Stale FTS5 index (triggers rebuild)
   - Missing DIRECTION.md (creates default template)
6. For non-fixable issues, explain what the developer needs to do manually:
   - Malformed frontmatter → developer needs to fix the YAML
   - Missing decision sections → developer needs to add the missing ## sections
   - Invalid `deprecates` reference → developer needs to correct the slug
   - Missing MCP servers → developer needs to configure them

## Rules
- Always present doctor results before suggesting fixes — let the developer see what's wrong
- Never run `cx doctor --fix` without the developer's approval
- If doctor reports zero issues, say so briefly — don't over-explain
- Doctor is read-only by default; `--fix` is the only mode that writes files
- If doctor cannot run (e.g., not in a cx-initialized project), tell the developer to run `cx init` first
