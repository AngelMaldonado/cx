# Skill: cx-memory

## Description
Teaches the Master and Contractor how to save observations, decisions, and session summaries using the cx memory commands. These agents use this skill during and at the end of work sessions to record team knowledge. Before saving, the agent consults DIRECTION.md to decide if the information is worth recording.

## Triggers
- Agent discovers something non-obvious (bug behavior, external system constraint, undocumented API behavior)
- Agent makes or helps make a technical decision (library choice, architecture pattern, convention)
- Agent is ending a work session or context is about to compact
- Developer explicitly asks to record something ("remember this", "save this decision", "log this")

## Steps
1. Before saving any observation or decision, read `docs/memories/DIRECTION.md`. Apply its rules:
   - If the information falls into an "always save" category → proceed
   - If it falls into a "never save" category → skip (tell the developer why if they asked)
   - If unclear → apply the threshold heuristics from DIRECTION.md
2. Determine the entity type:
   - **Observation**: something happened or was learned (use `cx memory save`)
   - **Decision**: a deliberate choice was made between alternatives (use `cx memory decide`)
   - **Session summary**: work session ending (use `cx memory session`)

### Saving an observation
3. Run: `cx memory save --title "<one-line summary>" --type <bugfix|discovery|pattern|context> --content "<full description>"`
4. Add optional flags if applicable:
   - `--change <name>` if working within an active change
   - `--files <path1,path2>` for referenced codebase files
   - `--specs <area1,area2>` for affected spec areas
   - `--tags <tag1,tag2>` for searchable tags
   - `--deprecates <slug>` if this replaces an outdated observation
5. Verify the command exits 0. If it fails, read the error output — it will list missing required fields.

### Recording a decision
6. Run: `cx memory decide --title "<what was decided>"`
7. The binary will prompt for or expect all five required sections: Context, Outcome, Alternatives, Rationale, Tradeoffs. Provide each as a flag or via stdin.
8. Add optional flags: `--change`, `--specs`, `--tags`, `--deprecates <slug>` (if this replaces an older decision or observation).
9. If `--deprecates` references another decision, the old decision's status will be set to `superseded` in the index.

### Saving a session summary
10. Run: `cx memory session --goal "<what you aimed to do>"`
11. Provide all sections: goal, accomplished items, discoveries, blockers (if any), files touched, and next steps.
12. The `## Next Steps` section is critical — it's what the primer uses for session recovery. Be specific: "Implement the retry logic in src/mqtt/client.go, starting from the publishWithRetry function" is better than "Continue MQTT work".

## Rules
- Always check DIRECTION.md before saving — never save noise
- Never save implementation details that are visible by reading the code (DIRECTION.md's default "never save" rule)
- Never save duplicate information that's already in an existing, non-deprecated observation
- For decisions: all five body sections are mandatory. Do not skip Alternatives or Tradeoffs — a decision without considered alternatives is not a decision
- For observations with `--deprecates`: verify the referenced slug exists (the binary validates this, but check first to avoid a failed command)
- Session summaries should be saved when ending work OR when context compaction is imminent — don't wait until the developer says "stop"
- Personal notes (cx memory note) are for cross-project personal preferences only — not for project observations
