# Delta Spec: doctor

Change: spec-system-evolution

## ADDED Requirements

### REQ-NEW-001: cx.yaml Validation Check
`cx doctor` MUST check `cx.yaml` if the file exists at the project root. The check MUST validate: (1) the file parses as valid YAML, and (2) the top-level structure contains only recognized fields (`schema`, `context`, `rules`). If `cx.yaml` is absent, no check is performed and no warning is emitted — the file is optional. If `cx.yaml` is present but malformed (YAML parse error or unrecognized top-level keys), `cx doctor` MUST emit a Warning (not Error). This check is not auto-fixable.

### REQ-NEW-002: cx.yaml Check in docs/ Structure Area
The `cx.yaml` validation check MUST appear within the "docs/ Structure Integrity" check area (check area 1), not as a new top-level area. The check row in the check table: `cx.yaml valid structure (if present)` | Warning | No.

## MODIFIED Requirements

### docs/ Structure Integrity Check Table
Previously: the docs/ Structure Integrity table contained 9 checks ending with orphan delta spec detection.
Now: the table gains one additional row: `cx.yaml` valid structure (if present) | Warning | No. This row is added after the orphan delta spec check.

## REMOVED Requirements

None. All existing `cx doctor` behaviors are preserved.
