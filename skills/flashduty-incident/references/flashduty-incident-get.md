# flashduty incident get

Get incident details. Single ID produces a vertical detail view; multiple IDs produce a table.

## Usage

```bash
flashduty incident get <id> [<id2> ...]
```

## Flags

No command-specific flags. Supports global flags (`--json`, `--no-trunc`, etc.).

## Output Columns

Single ID: vertical key-value detail view.
Multiple IDs: table with columns ID, TITLE, SEVERITY, PROGRESS, CHANNEL, CREATED.

## Examples

```bash
# Single incident - vertical detail view
flashduty incident get abc123

# Multiple incidents - table format
flashduty incident get abc123 def456 ghi789
```
