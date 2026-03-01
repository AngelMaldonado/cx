# Spec: Scenario Format

Scenarios define testable behaviors for spec areas using a Given/When/Then format. They complement `spec.md` files by providing concrete examples of how the system behaves in specific situations.

---

## When to write scenarios

Scenarios are **always optional**. `cx doctor` does not warn about missing `scenarios.md` files. They're useful when:

- A spec area has edge cases that are hard to capture in prose
- Multiple behaviors interact in non-obvious ways
- The team wants executable acceptance criteria before implementation
- A bug was found and the team wants to prevent regression

Scenarios are NOT useful when:
- The spec is straightforward enough that prose is sufficient
- The behavior is purely conceptual (e.g., design principles, architecture decisions)
- The spec area is still evolving rapidly (scenarios add friction to changes)

---

## File location

```
docs/specs/<area>/scenarios.md
```

One file per spec area, alongside `spec.md`.

---

## Format

```markdown
# Scenarios: <Spec Area Name>

## <Feature or Behavior Group>

### Scenario: <descriptive name>

**Given** <initial condition>
**When** <action or event>
**Then** <expected outcome>

### Scenario: <another scenario>

**Given** <initial condition>
**And** <additional condition>
**When** <action or event>
**Then** <expected outcome>
**And** <additional outcome>
```

### Syntax rules

- **Given/When/Then** must be bold (`**Given**`, `**When**`, `**Then**`)
- **And** extends the previous keyword (additional Given, When, or Then clause)
- Each scenario has exactly one **When** (the trigger)
- Scenarios are grouped under `##` headings by feature or behavior area
- Each scenario has a `###` heading with a descriptive name
- Scenario names should describe the situation, not the expected outcome: "expired token on protected endpoint" not "should return 401"

### Example

```markdown
# Scenarios: Memory

## Observation Deprecation

### Scenario: New observation deprecates an existing one

**Given** an observation "mqtt-256kb-limit" exists in docs/memories/observations/
**And** the observation is not currently deprecated
**When** `cx memory save --deprecates 2026-02-21T10-00-00-angel-mqtt-drops` is executed
**Then** the new observation file is written to docs/memories/observations/
**And** on next index rebuild, the old observation's `deprecated` flag is set to 1
**And** the old observation is excluded from `cx search` results by default

### Scenario: Deprecating a non-existent entity

**Given** no observation with slug "does-not-exist" exists
**When** `cx memory save --deprecates does-not-exist` is executed
**Then** the command exits with a non-zero status
**And** the error message says "referenced entity does not exist"
**And** no file is written

## Decision Status

### Scenario: Decision deprecates another decision

**Given** an active decision "rest-for-all" exists
**When** a new decision is created with `deprecates: rest-for-all`
**Then** the new decision file is written with `status: active`
**And** on next index rebuild, the old decision's `status` is set to `superseded` in the index
**And** the old decision's file is not modified

### Scenario: Cancelled decision

**Given** an active decision "offline-alert-thresholds" exists
**When** a new decision is created with `status: cancelled`
**And** the decision does not have a `deprecates` field
**Then** the cancelled decision is excluded from default context output
**And** the original decision is NOT affected (still active)
```

---

## Delta scenarios

When a change modifies a spec area that has scenarios, the change's delta should include updated or new scenarios:

```
docs/changes/<name>/specs/<area>/delta.md
```

The delta's ## ADDED section can include new scenarios. The ## MODIFIED section can reference existing scenarios by name. During archive merge, scenario changes are merged alongside spec changes.

---

## Relationship to tests

Scenarios are documentation, not executable tests. They describe expected behavior in human-readable form. Implementation teams can use them as the basis for writing automated tests, but CX does not execute scenarios or verify them against code.
