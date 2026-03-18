# Delta Spec: skills

Change: spec-system-evolution

## ADDED Requirements

### REQ-NEW-001: cx-change Skill — Mandatory cx instructions Call
The `cx-change` skill MUST mandate that the agent calls `cx instructions <artifact>` before filling any change document. This call MUST appear as the first numbered step in the skill's Steps section for artifact-filling workflows. The rule MUST also appear in the skill's Rules section: "Always call `cx instructions <artifact>` before writing proposal.md, specs, design.md, or tasks.md."

### REQ-NEW-002: cx-change Skill — New Commands Coverage
The `cx-change` skill MUST document the three new commands introduced by this change:
- `cx change verify <name>` — run before archive to scaffold the verification prompt
- `cx change spec-sync <name>` — merge delta specs into canonical specs without archiving
- `cx change archive <name> --skip-specs` — archive without delta spec validation or verify (for non-behavioral changes)

### REQ-NEW-003: Skill Registry — New Commands
The skill registry table in the skills spec MUST reflect that `cx-change` now covers five commands: `cx change new`, `cx change status`, `cx change archive`, `cx change verify`, and `cx change spec-sync`.

## MODIFIED Requirements

### cx-change Skill — Steps for Archive Workflow
Previously: the archive workflow in `cx-change` steps proceeded directly from tasks complete to `cx change archive <name>`.
Now: the archive workflow MUST include a verify step between tasks completion and archive: (1) run `cx change verify <name>`, (2) review output and write `verify.md` (or dispatch Reviewer agent), (3) then run `cx change archive <name>`.

### cx-change Skill — Rules
Previously: `cx-change` rules did not mention context gathering before artifact creation.
Now: a new rule MUST be added: "Always call `cx instructions <artifact>` before writing any change document. Do not skip this step even for simple artifacts."

## REMOVED Requirements

None. All existing skill protocol requirements and skill format rules are preserved.
