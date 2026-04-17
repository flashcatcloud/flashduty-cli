# flashduty postmortem list

List post-mortem reports with filtering.

## Usage

```bash
flashduty postmortem list [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--status` | string | | Filter: `drafting` or `published` |
| `--channel` | string | | Comma-separated channel IDs |
| `--team` | string | | Comma-separated team IDs |
| `--since` | string | | Created after (time filter) |
| `--until` | string | | Created before (time filter) |
| `--limit` | int | `20` | Max results |
| `--page` | int | `1` | Page number |

## Output Columns

ID, TITLE, STATUS, CHANNEL, CREATED.

## Examples

```bash
# Published post-mortems from the last month
flashduty postmortem list --status published --since 30d

# Post-mortems for specific teams
flashduty postmortem list --team 1,2,3
```
