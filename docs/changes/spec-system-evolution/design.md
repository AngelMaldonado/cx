# Design: spec-system-evolution

## Architecture

This change is primarily additive: new Go packages, new commands, new templates, and targeted modifications to existing archive logic. No existing packages are removed. The dependency graph among the new components is:

```
cx.yaml (project root)
    │
    └── internal/config/
            │
            └── internal/instructions/  ←── internal/templates/docs/spec.md (updated)
                    │                         internal/templates/docs/*.md (frontmatter)
                    └── cmd/instructions.go   (cx instructions <artifact>)

internal/verify/
    ├── BuildPrompt()  →  reads delta specs + canonical specs + proposal + design
    └── cmd/verify.go  →  cx change verify <name>

internal/change/change.go (modified)
    ├── Archive() → now gates on verify.md + handles --skip-specs
    ├── SpecSync() → new: partial merge without archive
    └── completeness check → updated to strip frontmatter before empty-template test
```

### New packages

**`internal/config/`**
Single exported function `Load(rootDir string) (*Config, error)`. Returns a zero-value `Config` struct if no `cx.yaml` is present — callers do not need to handle absent config as an error. The `Config` struct fields: `Schema string`, `Context string`, `Rules map[string][]string`.

**`internal/instructions/`**
Two files:
- `graph.go` — defines the static `ArtifactGraph` (proposal, specs, design, tasks as `Artifact` structs with `ID`, `File`, `Requires`, and `Unlocks` fields). Exports `DependenciesOf(artifact string) []string` and `UnlocksOf(artifact string) []string`.
- `instructions.go` — exports `Build(rootDir, artifact string) (string, error)`. Reads `internal/config.Load()`, the appropriate embedded template, `docs/specs/index.md`, and the graph. Returns a formatted multi-section string for stdout.

**`internal/verify/`**
Two exported functions:
- `BuildPrompt(rootDir, changeName string) (string, error)` — reads all delta specs for the change, the corresponding canonical specs, and the change's proposal.md and design.md. Extracts REQ-NNN lines from delta specs to populate the COMPLETENESS checklist. Returns a structured verification prompt string.
- `Record(rootDir, changeName, result string) error` — writes a pre-filled `verify.md` stub to `docs/changes/<name>/verify.md`. The agent fills the result content; this function creates the file with frontmatter.

### Modified packages

**`internal/change/change.go`**
- `ArchiveOptions` struct added as parameter to `Archive()`. Fields: `SkipSpecs bool`.
- New verify gate in `Archive()`: if `!opts.SkipSpecs`, check that `verify.md` exists and contains "PASS" or has no lines matching "CRITICAL". Block with a clear error if absent or FAIL.
- `ChangeInfo` struct gains: `HasVerify bool`, `VerifyStatus string` (PENDING/PASS/FAIL), `DeltasSynced []string`.
- Completeness check updated: strip YAML frontmatter block (everything between opening `---` and closing `---`) before testing if remaining content equals the template stub. A file with only frontmatter and whitespace is treated as empty.

### New templates

- `internal/templates/docs/cx.yaml` — commented template with `schema`, `context`, and `rules` fields
- `internal/templates/docs/verify.md` — frontmatter stub with Result, Completeness, Correctness, Coherence, Issues sections

### Updated templates

All six existing artifact templates (`spec.md`, `delta-spec.md`, `proposal.md`, `design.md`, `tasks.md`, `masterfile.md`) drop their H1 title line and open with YAML frontmatter. The `spec.md` template gains significant structure (Purpose paragraph, REQ-NNN naming, Scenarios pointer, RFC 2119 keywords). The `delta-spec.md` template gains three Scenarios sections (ADDED/MODIFIED/REMOVED).

## Technical Decisions

**No binary enforcement of fill order during implementation.** The dependency graph is enforced only at `cx change archive` time, not during artifact creation. `cx change status` shows BLOCKED/READY/IN-PROGRESS/DONE as advisory information. This avoids interrupting agent workflows where design and proposal are written together.

**`cx.yaml` is not a schema constraint — it is agent guidance.** The dependency graph is a hardcoded Go data structure, not a `cx.yaml` concern. `cx.yaml` exists solely to inject project context and artifact-generation rules into `cx instructions` output. Agents never need to find or parse `cx.yaml` themselves.

**`cx instructions` replaces ad-hoc multi-file context gathering.** The command consolidates: template text, project context, project rules for the artifact, dependency state, and existing spec index. One call. Agents call this at the start of filling any change artifact. The cx-change skill is updated to mandate this.

**`--skip-specs` replaces progressive rigor modes.** Two rigor modes (lite/full) were considered and dropped. A single per-invocation flag is simpler to reason about and covers the practical escape hatch for non-behavioral changes. The skip is logged in the archive memory save for auditability.

**Verify is agent-driven, binary-scaffolded.** The binary cannot assess whether Go code satisfies a spec requirement. Its role is to structure the verification prompt (extract requirements, format checklist dimensions) and record the result. The Reviewer agent performs the actual review. `cx change archive` gates on `verify.md` existence and PASS state.

**Frontmatter SYNCED marker on delta files.** After `cx change spec-sync`, the binary writes `synced: true` into the delta file's YAML frontmatter. The archive merge reads this marker and skips already-synced deltas, preventing double-merge.

**Completeness check must be frontmatter-aware.** The existing check (non-empty content != template text) breaks when templates shift from H1 to frontmatter. The updated check strips the YAML front block before comparing. A file with only frontmatter is treated as template-only (empty state).

**REQ-NNN naming is convention, not enforcement.** The template signals the naming pattern; agents adopt it. No binary parsing or validation of REQ-NNN format. The convention enables stable delta merge references (requirement names rather than line numbers), but the system functions without it.

## Implementation Notes

**Order of implementation:**
1. Update all templates (no Go changes, validates the template designs are sound)
2. Add `internal/config/` package and tests
3. Add `internal/instructions/` package and tests (depends on config)
4. Add `cx instructions` command
5. Update completeness check in `internal/change/change.go` (isolated change, easy to test)
6. Add `internal/verify/` package and tests
7. Add `cx change verify` subcommand
8. Add `cx change spec-sync` subcommand
9. Update `cx change archive` (add verify gate, `--skip-specs` flag)
10. Update `cx change status` (add computed state, verify state)
11. Update `cx init` to create `cx.yaml`
12. Update `cx doctor` to validate `cx.yaml` if present
13. Update cx-change skill to mandate `cx instructions` call

**`cx instructions` output when index.md missing:** Print `(no specs found — run cx init)` rather than failing. The command is called early in new projects before specs exist.

**Verify PASS detection:** `Archive()` reads `verify.md` and checks: (a) file contains the string "PASS", and (b) file does not contain a line that starts with "CRITICAL". Both conditions must hold. A file that says "PASS" with a CRITICAL issue noted is treated as FAIL.

**`cx change spec-sync` merge flow:** Identical to the archive's step 2 (prints canonical + delta to stdout, agent produces merged spec, developer approves, agent writes result). After writing, the binary sets `synced: true` in the delta's frontmatter using the same YAML frontmatter parser as `internal/config`.

**`cx doctor` cx.yaml check:** Warning (not error) if `cx.yaml` exists but fails YAML parse or has an unrecognized top-level key. No warning if `cx.yaml` is absent — it is optional.
