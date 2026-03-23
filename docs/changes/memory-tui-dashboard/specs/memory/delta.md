---
change: memory-tui-dashboard
spec: memory
type: delta
---

# Delta: memory — memory-tui-dashboard

## Added

### `cx memory deprecate <id>` command

New subcommand added to the memory command group. Sets `deprecated=1` on an existing memory row in `.cx/memory.db`. Errors if the ID is not found (non-zero exit). Exposed as a CLI command (not only a TUI action) so it can be called from scripts or other tooling.

Added to the Command Reference table under a new **Deprecation** section.

### TUI Dashboard section

New top-level section added to the memory spec documenting `cx dashboard` (aliases: `dash`, `ui`) as a direct developer entry point. Documents:

- The 8 navigable views and their tab number assignments
- Read/write access model (read-only except sync push/pull and memory deprecation)
- 5-second polling refresh cycle
- Responsive layout thresholds (80+ cols two-pane, <80 single-pane, <40×10 too-small guard)
- Full keyboard shortcut reference for global keys, per-view navigation, and the detail overlay

## No Removals

No existing spec content was removed. The new `cx memory deprecate` command does not replace any existing deprecation mechanism — it surfaces the existing `deprecated=1` DB flag as an explicit CLI action.
