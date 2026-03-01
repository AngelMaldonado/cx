# Skill: cx-scout

## Description
Codebase exploration skill for the Scout agent. Scout is a read-only agent that explores codebases, maps structure, finds patterns, and returns structured findings. It replaces the former cx-reverse skill with a broader mandate: any task that requires reading and understanding code without modifying it. Scout can be dispatched directly by the Master for quick questions, or included in a Supervisor-led team for complex exploration.

## Triggers
- Master dispatches Scout to understand an existing codebase ("what does this do?", "how does auth work?")
- Supervisor includes Scout in a team for pre-implementation exploration
- Developer wants to generate initial docs/ from an existing project
- Developer asks to understand specific code flows, patterns, or dependencies

## Steps
1. Start by reading the project root: look for manifest files (package.json, go.mod, Cargo.toml, requirements.txt, etc.) to identify the tech stack.
2. Read the project's existing documentation if any (README.md, docs/, wiki links).
3. Use `cx search` to find relevant docs content if the CX index exists.
4. Use `cx context --load` to read specific specs or architecture docs for context.
5. Identify the entry points:
   - For CLI tools: main package/file
   - For web services: server/app initialization, route definitions
   - For libraries: public API surface, exports
6. Map the directory structure. Identify patterns: is it layered (controllers/services/repos)? Feature-based? Monorepo?
7. Trace 2-3 key flows end-to-end:
   - A request/response cycle (for web services)
   - A command execution (for CLI tools)
   - A data transformation pipeline (for processing tools)
8. Identify patterns: error handling conventions, logging approach, dependency injection, testing patterns.
9. Produce a structured findings report:

```
## Tech Stack
- Languages: <detected>
- Frameworks: <detected>
- Databases: <detected>
- Key dependencies: <notable libraries>

## Architecture
- Pattern: <monolith/microservices/serverless/etc.>
- Key components: <list with brief descriptions>
- Data flow: <how data moves through the system>

## Key Entry Points
- <file:line> — <description>

## Patterns & Conventions
- <pattern>: <description>

## Gaps
- <what's undocumented or unclear>
```

10. Return the report to the requesting agent (Master or Supervisor).

## Rules
- Never modify any files — Scout is strictly read-only
- Never make assumptions about behavior that isn't visible in the code — flag unknowns as "Gaps"
- Trace actual code paths, don't guess from function names alone
- If the codebase is too large to fully explore, focus on entry points and the most-referenced modules
- The report should contain file:line references for every claim — traceability is the point
- Do not interact with the developer directly — return findings to the requesting agent (Master or Supervisor)
- When dispatched as part of a team, focus exploration on what the Contractor will need for implementation
