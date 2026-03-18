---
name: spec-system-evolution
type: tasks
---

## Task Breakdown

All tasks are assigned to the `general-purpose` executor agent (no project-specific executors defined).

---

## Batch 1: Templates (no Go changes, foundational for all other batches)

### TASK-01: Update artifact templates to frontmatter format

**Agent:** general-purpose

**What:** Replace H1 title lines in all six existing artifact templates with YAML frontmatter blocks. No content structure changes beyond the header swap.

**Files:**
- `internal/templates/docs/proposal.md` — remove `# Proposal: {{name}}`, add `---\nname: {{name}}\ntype: proposal\n---`
- `internal/templates/docs/design.md` — remove `# Design: {{name}}`, add `---\nname: {{name}}\ntype: design\n---`
- `internal/templates/docs/tasks.md` — remove `# Tasks: {{name}}`, add `---\nname: {{name}}\ntype: tasks\n---`
- `internal/templates/docs/masterfile.md` — remove `# Masterfile: {{name}}`, add `---\nname: {{name}}\ntype: masterfile\n---`
- `internal/templates/docs/delta-spec.md` — remove `# Delta Spec: {{area}}` and the bare `Change: {{name}}` line, add frontmatter with `name`, `area`, `type: delta-spec`, `change` fields. Add three new sections after REMOVED Requirements: `## ADDED Scenarios`, `## MODIFIED Scenarios`, `## REMOVED Scenarios`.
- `internal/templates/docs/spec.md` — remove `# Spec: {{name}}`, add frontmatter. Expand from the thin 3-section stub to include: a Purpose paragraph, REQ-NNN named requirements subsections with RFC 2119 keywords, and a Scenarios section pointing to an optional `scenarios.md` file.

**Notes:** The embed.go uses `//go:embed agents/*.md subagents/*.md docs/*.md` — no embed.go change needed for these files. These templates are loaded via `templates.MustContent("docs/xxx.md")` in `internal/change/templates.go`.

**Acceptance:** All six template files open with valid YAML frontmatter. spec.md demonstrates the REQ-NNN pattern. delta-spec.md has three Scenarios sections.

---

### TASK-02: Add cx.yaml and verify.md templates

**Agent:** general-purpose

**What:** Create two new template files under `internal/templates/docs/`. Also update embed.go to cover the new .yaml file.

**Files to create/modify:**
- `internal/templates/docs/cx.yaml` (new) — commented template with `schema: cx/v1`, `context: ""`, and `rules: {}` fields. Include inline YAML comments explaining each field's purpose.
- `internal/templates/docs/verify.md` (new) — frontmatter stub with `name` and `type: verify` fields, followed by five sections: `## Result`, `## Completeness`, `## Correctness`, `## Coherence`, `## Issues`.
- `internal/templates/embed.go` (modify) — change the embed directive from `//go:embed agents/*.md subagents/*.md docs/*.md` to `//go:embed agents/*.md subagents/*.md docs/*.md docs/*.yaml` so cx.yaml is included in the embedded FS.

**Acceptance:** `templates.Content("docs/cx.yaml")` and `templates.Content("docs/verify.md")` return content without error after the embed change.

---

## Batch 2: New Go packages (foundation for new commands)

### TASK-03: Add internal/config package

**Agent:** general-purpose

**What:** Create `internal/config/config.go` with a `Config` struct and `Load()` function.

**Files to create:**
- `internal/config/config.go`
- `internal/config/config_test.go`

**Spec:**
```go
package config

type Config struct {
    Schema  string              // "cx/v1"
    Context string              // free-form project context
    Rules   map[string][]string // artifact name → list of rule strings
}

// Load reads cx.yaml from rootDir. Returns zero-value Config (nil error) if
// cx.yaml is absent. Returns non-nil error on YAML parse failure or if
// unrecognized top-level keys are present.
func Load(rootDir string) (*Config, error)
```

Use `gopkg.in/yaml.v3` (check go.mod for the import path already in use). Return `&Config{}` when file is absent. Validate top-level keys are only `schema`, `context`, `rules`; return a descriptive error for unrecognized keys so the doctor check (TASK-15) can emit a Warning.

**Test cases:**
- File absent → zero-value Config, nil error
- File present with all fields → correct struct values
- File present with unknown top-level key → non-nil error
- File present but invalid YAML → non-nil error

---

### TASK-04: Add internal/instructions package

**Agent:** general-purpose

**What:** Create `internal/instructions/` with two files: `graph.go` and `instructions.go`.

**Files to create:**
- `internal/instructions/graph.go`
- `internal/instructions/instructions.go`
- `internal/instructions/instructions_test.go`

**graph.go spec:**
```go
package instructions

type Artifact struct {
    ID       string   // "proposal", "specs", "design", "tasks"
    File     string   // relative file path within change dir
    Requires []string // IDs that must be DONE before this
    Unlocks  []string // IDs this enables
}

var ArtifactGraph = []Artifact{
    {ID: "proposal", File: "proposal.md", Requires: []string{},                    Unlocks: []string{"specs", "design"}},
    {ID: "specs",    File: "specs/",      Requires: []string{"proposal"},            Unlocks: []string{"tasks"}},
    {ID: "design",   File: "design.md",   Requires: []string{"proposal"},            Unlocks: []string{"tasks"}},
    {ID: "tasks",    File: "tasks.md",    Requires: []string{"specs", "design"},     Unlocks: []string{"verify"}},
}

func DependenciesOf(artifact string) []string
func UnlocksOf(artifact string) []string
```

**instructions.go spec:**
```go
// Build returns a formatted multi-section string for stdout. Sections:
// 1. Artifact template text
// 2. Project context from cx.yaml (if present)
// 3. Project rules for this artifact from cx.yaml rules map
// 4. Dependency requirements from ArtifactGraph
// 5. What this artifact unlocks
// 6. Existing spec index (docs/specs/index.md), or fallback message if absent
func Build(rootDir, artifact string) (string, error)
```

Calls `config.Load(rootDir)` for project context and rules. If artifact name not in ArtifactGraph, returns an error. If `docs/specs/index.md` absent, uses the string `(no specs found — run cx init)`. Reads artifact template via `templates.Content("docs/<artifact>.md")`.

**Test cases:** Build with no cx.yaml (zero config), Build with cx.yaml rules for the artifact, Build with missing index.md (uses fallback), Build with invalid artifact name (returns error).

**Depends on:** TASK-01 (templates), TASK-02 (embed), TASK-03 (config.Load).

---

### TASK-05: Add internal/verify package

**Agent:** general-purpose

**What:** Create `internal/verify/` with two exported functions.

**Files to create:**
- `internal/verify/verify.go`
- `internal/verify/verify_test.go`

**Spec:**
```go
package verify

// BuildPrompt reads all delta specs for the change, extracts REQ-NNN lines
// to form a COMPLETENESS checklist, and returns a structured verification
// prompt string. Also reads proposal.md and design.md for intent context.
func BuildPrompt(rootDir, changeName string) (string, error)

// Record writes a pre-filled verify.md stub to docs/changes/<name>/verify.md.
// Uses the embedded verify.md template. Skips (no error) if file already exists.
func Record(rootDir, changeName string) error
```

`BuildPrompt`: reads `docs/changes/<changeName>/specs/*/spec.md`, extracts lines matching `### REQ-[A-Z0-9-]+:` as checklist items, reads `docs/changes/<changeName>/proposal.md` and `design.md`. Returns a prompt with four sections: COMPLETENESS (checklist of REQ-NNN items), CORRECTNESS (implementation correctness dimensions), COHERENCE (consistency with proposal intent), and Instructions.

`Record`: uses `templates.Content("docs/verify.md")` and the `atomicWrite` pattern (see `internal/change/change.go` for the pattern). Replaces `{{name}}` in template. Does not overwrite if file exists.

**Test cases:** BuildPrompt with no delta specs (returns prompt with empty checklist), BuildPrompt extracts REQ-NNN lines correctly, Record creates file with frontmatter, Record is idempotent on second call.

**Depends on:** TASK-02 (verify.md template).

---

## Batch 3: Modify internal/change package

### TASK-06: Update ChangeInfo, Archive, and completeness check in internal/change/change.go

**Agent:** general-purpose

**What:** Three targeted changes to `internal/change/change.go`: frontmatter-aware `fileModified`, extended `ChangeInfo`, and `ArchiveOptions` with verify gate.

**Files to modify:**
- `internal/change/change.go`

**Change 1 — frontmatter-aware fileModified:**
Update `fileModified(dir, filename, template string) bool` to strip the YAML frontmatter block before comparing content to template. Logic: if file content starts with `---\n`, find the next occurrence of `\n---\n` (or `\n---` at EOF) and remove everything up to and including the closing `---` line. Compare only the remaining content (trimmed) against the template (trimmed). A file with only frontmatter and whitespace is treated as unmodified.

**Change 2 — extended ChangeInfo struct:**
```go
type ChangeInfo struct {
    Name         string
    Path         string
    HasProposal  bool
    HasDesign    bool
    HasTasks     bool
    HasVerify    bool
    VerifyStatus string   // "PENDING", "PASS", "FAIL"
    DeltaSpecs   []string
    SyncedDeltas []string // areas where delta frontmatter has synced: true
}
```

Update `ListChanges()` to populate the new fields: read `verify.md` if present, determine VerifyStatus (PASS: contains "PASS" AND no line starting with "CRITICAL"; FAIL: file present but not PASS; PENDING: file absent). Read each delta's frontmatter to detect `synced: true`.

**Change 3 — ArchiveOptions and verify gate:**
```go
type ArchiveOptions struct {
    SkipSpecs bool
}

func Archive(rootDir, name string, opts ArchiveOptions) (*ArchiveResult, error)
```
In `Archive()`, after the existing completeness check and before the bootstrap-missing-specs step: if `!opts.SkipSpecs`, check that `verify.md` exists and has VerifyStatus PASS. Return a clear error if not. Also: skip delta areas with `synced: true` in their frontmatter during the bootstrap step (those are already merged into canonical specs).

**Test cases:** fileModified returns false for frontmatter-only file, fileModified returns true when body content differs, Archive blocks on missing verify.md, Archive allows --skip-specs, Archive with PASS verify.md succeeds, synced deltas are skipped in bootstrap.

---

### TASK-07: Add SpecSync function to internal/change/change.go

**Agent:** general-purpose

**What:** Add `SpecSync()` to `internal/change/change.go`. Separated from TASK-06 to allow independent implementation with no conflict risk.

**Files to modify:**
- `internal/change/change.go`

**Spec:**
```go
type SpecSyncResult struct {
    Areas  []string // delta spec areas found
    Prompt string   // merge prompt for stdout
}

// SpecSync generates the agent-assisted merge prompt for each unsynced delta
// spec and marks merged ones as synced: true in their frontmatter.
// Requires only proposal.md to be filled.
func SpecSync(rootDir, name string) (*SpecSyncResult, error)
```

Implementation:
1. Check proposal.md is filled using the frontmatter-aware `fileModified`.
2. Find all delta areas under `docs/changes/<name>/specs/*/spec.md`.
3. For each area, read the delta spec and the canonical spec (if it exists). Skip areas where delta frontmatter already has `synced: true`.
4. Build a merge prompt string (print canonical + delta side by side with clear markers).
5. Set `synced: true` in each delta file's YAML frontmatter using a simple string manipulation (insert `synced: true` into the frontmatter block before the closing `---`).
6. Return `SpecSyncResult` with the prompt and list of processed areas.

Note: The binary structures the prompt; the agent performs the actual merge write. The `synced: true` marker is set after the merge is confirmed by the calling command.

**Depends on:** TASK-06 (frontmatter-aware fileModified must be available).

---

### TASK-08: Add VerifyTemplate to internal/change/templates.go

**Agent:** general-purpose

**What:** Add a `VerifyTemplate(name string) string` function to `internal/change/templates.go`.

**Files to modify:**
- `internal/change/templates.go`

**Change:**
```go
func VerifyTemplate(name string) string {
    tmpl := templates.MustContent("docs/verify.md")
    return strings.ReplaceAll(tmpl, "{{name}}", name)
}
```

Also verify that `DeltaSpecTemplate` still functions correctly after the frontmatter change to `delta-spec.md` from TASK-01 (the `{{name}}` and `{{area}}` placeholders must remain in the updated template).

**Depends on:** TASK-02 (verify.md template).

---

## Batch 4: New and modified commands

### TASK-09: Add cx instructions command

**Agent:** general-purpose

**What:** Create `cmd/instructions.go` and register the command in `cmd/root.go`.

**Files to create/modify:**
- `cmd/instructions.go` (new)
- `cmd/root.go` (add `instructionsCmd` to `init()`)

**Spec:**
```go
var instructionsCmd = &cobra.Command{
    Use:   "instructions <artifact>",
    Short: "Get template, context, and dependency info for a change artifact",
    Args:  cobra.ExactArgs(1),
    RunE:  runInstructions,
}
```

`runInstructions`:
1. Calls `project.IsGitRepo()` for rootDir.
2. Validates artifact arg is one of `proposal`, `specs`, `design`, `tasks`; if not, prints error listing valid names and returns `errExitCode1`.
3. Calls `instructions.Build(rootDir, artifact)`.
4. Prints result to stdout with `fmt.Print` (raw output, not ui helpers — agents consume this directly).
5. On error: `ui.PrintError` and return `errExitCode1`.

In `cmd/root.go` `init()`: add `rootCmd.AddCommand(instructionsCmd)`.

**Depends on:** TASK-04 (internal/instructions).

---

### TASK-10: Add cx change verify subcommand

**Agent:** general-purpose

**What:** Add `changeVerifyCmd` cobra command to `cmd/change.go` and wire in the `init()` function.

**Files to modify:**
- `cmd/change.go`

**Spec:**
```go
var changeVerifyCmd = &cobra.Command{
    Use:   "verify <name>",
    Short: "Scaffold verification prompt for a change before archiving",
    Args:  cobra.ExactArgs(1),
    RunE:  runChangeVerify,
}
```

`runChangeVerify`:
1. Calls `project.IsGitRepo()`.
2. Calls `verify.BuildPrompt(rootDir, name)` — prints the structured prompt to stdout with `fmt.Print`.
3. Calls `verify.Record(rootDir, name)` to create the verify.md stub (idempotent).
4. Prints `ui.PrintMuted(fmt.Sprintf("verify.md created at docs/changes/%s/verify.md — fill it in, then run cx change archive %s", name, name))`.
5. On error: `ui.PrintError` and return `errExitCode1`.

Add `changeCmd.AddCommand(changeVerifyCmd)` in `init()`.

**Depends on:** TASK-05 (internal/verify).

---

### TASK-11: Add cx change spec-sync subcommand

**Agent:** general-purpose

**What:** Add `changeSpecSyncCmd` to `cmd/change.go`.

**Files to modify:**
- `cmd/change.go`

**Spec:**
```go
var changeSpecSyncCmd = &cobra.Command{
    Use:   "spec-sync <name>",
    Short: "Merge delta specs into canonical specs without archiving",
    Args:  cobra.ExactArgs(1),
    RunE:  runChangeSpecSync,
}
```

`runChangeSpecSync`:
1. Calls `project.IsGitRepo()`.
2. Calls `change.SpecSync(rootDir, name)` — returns `SpecSyncResult` with merge prompt and areas.
3. Prints result.Prompt to stdout with `fmt.Print` (raw, for agent consumption).
4. Prints `ui.PrintSuccess(fmt.Sprintf("spec-sync complete — %d delta(s) marked synced", len(result.Areas)))`.
5. On error: `ui.PrintError` and return `errExitCode1`.

Add `changeCmd.AddCommand(changeSpecSyncCmd)` in `init()`.

**Depends on:** TASK-07 (SpecSync in internal/change).

---

### TASK-12: Update cx change archive subcommand

**Agent:** general-purpose

**What:** Add `--skip-specs` flag to `changeArchiveCmd` and update the `Archive()` call to pass `ArchiveOptions`.

**Files to modify:**
- `cmd/change.go`

**Changes:**
- Add `var skipSpecsFlag bool` at package scope.
- In the `init()` function: `changeArchiveCmd.Flags().BoolVar(&skipSpecsFlag, "skip-specs", false, "Skip delta spec validation and verify gate (for non-behavioral changes)")`.
- In `runChangeArchive`: construct `opts := change.ArchiveOptions{SkipSpecs: skipSpecsFlag}` and call `change.Archive(rootDir, name, opts)`.
- When `skipSpecsFlag` is true and archive succeeds, append a muted line: `ui.PrintMuted("  skipped spec verification (--skip-specs)")`.

**Depends on:** TASK-06 (ArchiveOptions struct and updated Archive signature).

---

### TASK-13: Update cx change status display

**Agent:** general-purpose

**What:** Update `runChangeStatus` in `cmd/change.go` to show verify state, synced delta markers, and advisory artifact states.

**Files to modify:**
- `cmd/change.go`

**Changes in `runChangeStatus`:**
- After the existing `proposal / design / tasks` line, add a verify state line: `fmt.Printf("    verify: %s\n", c.VerifyStatus)`.
- For delta specs list, mark synced areas: iterate `c.DeltaSpecs`, append `[synced]` suffix for those in `c.SyncedDeltas`.
- Compute per-artifact advisory state using `instructions.DependenciesOf()` and the `Has*` bools:
  - proposal: DONE if HasProposal, else READY (no deps)
  - design: DONE if HasDesign; READY if HasProposal; else BLOCKED
  - tasks: DONE if HasTasks; READY if HasProposal && HasDesign; else BLOCKED
  - verify: DONE if VerifyStatus==PASS; READY if HasTasks; else BLOCKED
- Print the advisory states alongside the existing checkmarks.
- Import `"github.com/amald/cx/internal/instructions"`.

**Depends on:** TASK-06 (ChangeInfo.VerifyStatus, SyncedDeltas), TASK-04 (ArtifactGraph and DependenciesOf).

---

## Batch 5: Init, Doctor, and Skills updates

### TASK-14: Update cx init to create cx.yaml

**Agent:** general-purpose

**What:** Insert a new step between DIRECTION.md setup (current step 5) and git hook installation (current step 6) in `runInit`.

**Files to modify:**
- `cmd/init.go`

**Changes:**
- After the DIRECTION.md block (the `ui.Pause(300 * time.Millisecond)` after DIRECTION.md), add:
  ```go
  // Step 6: cx.yaml
  cxYamlPath := filepath.Join(rootDir, "cx.yaml")
  if _, err := os.Stat(cxYamlPath); os.IsNotExist(err) {
      content, _ := templates.Content("docs/cx.yaml")
      if err := os.WriteFile(cxYamlPath, []byte(content), 0o644); err != nil {
          ui.PrintWarning(fmt.Sprintf("writing cx.yaml: %v", err))
      } else {
          ui.PrintSuccess("cx.yaml    project config created")
      }
  } else {
      ui.PrintMuted("skipped cx.yaml (exists)")
  }
  ui.Pause(200 * time.Millisecond)
  ```
- Add import `"github.com/amald/cx/internal/templates"`.
- Renumber subsequent step comments (git hooks → Step 7, register → Step 8, API keys/MCP → Step 9, first-time prefs → Step 10, summary → Step 11).
- In the summary block at the bottom, add `ui.PrintItem("config", "cx.yaml")` after the direction item.

**Depends on:** TASK-02 (cx.yaml template embedded).

---

### TASK-15: Update cx doctor to validate cx.yaml

**Agent:** general-purpose

**What:** Add a cx.yaml validation check to `CheckDocsStructure` in `internal/doctor/checks.go`.

**Files to modify:**
- `internal/doctor/checks.go`

**Change:** After the DIRECTION.md result block in `CheckDocsStructure`, add:
```go
// Check cx.yaml structure if present (optional file)
cxYamlPath := filepath.Join(rootDir, "cx.yaml")
if _, statErr := os.Stat(cxYamlPath); statErr == nil {
    _, loadErr := config.Load(rootDir)
    if loadErr != nil {
        group.Results = append(group.Results, CheckResult{
            Name:     "cx.yaml",
            Severity: Warning,
            Message:  fmt.Sprintf("cx.yaml: %v", loadErr),
            Fixable:  false,
        })
    } else {
        group.Results = append(group.Results, CheckResult{
            Name:     "cx.yaml",
            Severity: Pass,
            Message:  "cx.yaml valid structure",
        })
    }
}
// if file absent: no check, no warning (cx.yaml is optional)
```

Add import `"github.com/amald/cx/internal/config"`.

**Depends on:** TASK-03 (internal/config, whose Load returns error for unrecognized keys).

---

### TASK-16: Update cx-change skill

**Agent:** general-purpose

**What:** Update `internal/skills/data/cx-change.md` to mandate `cx instructions`, document new commands, and update the archive workflow steps.

**Files to modify:**
- `internal/skills/data/cx-change.md`

**Changes:**

In the Steps section, insert as Step 1:
```
1. Before filling any change document, run `cx instructions <artifact>` to receive the template, project context, dependency state, and spec index.
```
Renumber existing steps 1→2, 2→3, 3→4, 4→5, 5→6.

Replace or extend step 6 ("Run cx change status") with the full archive workflow:
```
6. Run `cx change verify <name>` once implementation is complete. Review the output and write `verify.md` (or dispatch the Reviewer agent to fill it).
7. Run `cx change archive <name>` to archive the completed change.
   - For non-behavioral changes (CI, docs, tooling): `cx change archive <name> --skip-specs`
8. For long-running changes needing early spec stabilization, run `cx change spec-sync <name>` before archiving.
```

In the Rules section, add:
```
- Always call `cx instructions <artifact>` before writing proposal.md, specs, design.md, or tasks.md. Do not skip this step even for simple artifacts.
```

Add a new Commands section documenting the three new commands with one-line descriptions.

**Notes:** The cx-change.md skill file is embedded by `internal/agents` and written to agent skill directories by `agents.WriteSkills()`. Changing the embedded source file is sufficient; no Go code changes are required.

---

## Dependency Order for Execution

```
TASK-01 (artifact templates)  ─┐
TASK-02 (cx.yaml, verify.md)  ─┤ Batch 1 — do first, no deps
                                │
          ┌─────────────────────┘
          │
TASK-03 (internal/config)       — after TASK-02
TASK-04 (internal/instructions) — after TASK-01, TASK-02, TASK-03
TASK-05 (internal/verify)       — after TASK-02
TASK-06 (change.go: ChangeInfo) — after TASK-01 (frontmatter format known)
TASK-07 (change.go: SpecSync)   — after TASK-06 (needs fileModified)
TASK-08 (templates.go: Verify)  — after TASK-02

TASK-09 (cmd/instructions)      — after TASK-04
TASK-10 (cmd change verify)     — after TASK-05, TASK-08
TASK-11 (cmd change spec-sync)  — after TASK-07
TASK-12 (cmd change archive)    — after TASK-06
TASK-13 (cmd change status)     — after TASK-06, TASK-04

TASK-14 (cmd/init.go)           — after TASK-02
TASK-15 (internal/doctor)       — after TASK-03
TASK-16 (skills)                — independent, any time
```

**Recommended execution batches:**
1. TASK-01, TASK-02 (templates — parallelizable)
2. TASK-03, TASK-05, TASK-06, TASK-16 (packages and skills — parallelizable after Batch 1)
3. TASK-04, TASK-07, TASK-08 (packages that depend on Batch 2 — parallelizable)
4. TASK-09, TASK-10, TASK-11, TASK-12, TASK-14, TASK-15 (commands — parallelizable after their package deps)
5. TASK-13 (status display — needs TASK-04 and TASK-06 both done)
