# Skill: cx-reverse

## Description
Reverse-engineering skill for a dedicated subagent. Spawned when the team needs to understand an existing, undocumented codebase. The subagent explores the codebase using its native file tools (read, glob, grep) and the cx binary's search/load capabilities, then produces a structured findings report for the main agent.

## Triggers
- Developer asks to understand an existing codebase ("reverse engineer this", "what does this codebase do?", "help me understand this project")
- Main agent encounters an unfamiliar codebase during onboarding
- Developer wants to generate initial docs/ from an existing project

## Steps
1. Start by reading the project root: look for manifest files (package.json, go.mod, Cargo.toml, requirements.txt, etc.) to identify the tech stack.
2. Read the project's existing documentation if any (README.md, docs/, wiki links).
3. Identify the entry points:
   - For CLI tools: main package/file
   - For web services: server/app initialization, route definitions
   - For libraries: public API surface, exports
4. Map the directory structure. Identify patterns: is it layered (controllers/services/repos)? Feature-based? Monorepo?
5. Trace 2-3 key flows end-to-end:
   - A request/response cycle (for web services)
   - A command execution (for CLI tools)
   - A data transformation pipeline (for processing tools)
6. Use `cx search` to find patterns across the codebase if docs/ already exists and is indexed.
7. Use `cx context --load spec <area>` to read any existing spec documentation.
8. Produce a structured findings report:

```
## Reverse Engineering Findings

### Tech Stack
- Language: <lang> (<version if detectable>)
- Framework: <framework>
- Database: <db>
- Key dependencies: <list>

### Architecture
- Pattern: <layered | feature-based | monorepo | etc.>
- Entry points: <list with file paths>
- Directory structure summary

### Key Flows
#### Flow 1: <name>
<step-by-step trace with file:line references>

#### Flow 2: <name>
<step-by-step trace with file:line references>

### Domain Model
- Core entities: <list with relationships>
- Data storage: <how data is persisted>

### Gaps & Risks
- Undocumented behavior: <list>
- Missing tests: <areas>
- Potential issues: <list>
```

9. Return this report to the main agent. The main agent uses it to populate docs/ (overview.md, architecture/index.md, initial specs).

## Rules
- Never modify any source code files — the reverse-engineering subagent is read-only
- Never make assumptions about behavior that isn't visible in the code — flag unknowns as "Gaps"
- Trace actual code paths, don't guess from function names alone
- If the codebase is too large to fully explore, focus on entry points and the most-referenced modules
- The report should contain file:line references for every claim — traceability is the point
- Do not interact with the developer directly — return findings to the main agent
