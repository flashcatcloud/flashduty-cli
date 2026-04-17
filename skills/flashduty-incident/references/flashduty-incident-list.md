# flashduty incident list

List incidents with filtering and pagination.

## Usage

```bash
flashduty incident list [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--progress` | string | | Filter by state: `Triggered`, `Processing`, `Closed` |
| `--severity` | string | | Filter by severity: `Critical`, `Warning`, `Info` |
| `--channel` | int | | Filter by channel ID |
| `--title` | string | | Search by title keyword |
| `--since` | string | `24h` | Start time (duration like `24h`, date, datetime, or unix timestamp) |
| `--until` | string | `now` | End time |
| `--limit` | int | `20` | Max results (max 100) |
| `--page` | int | `1` | Page number |

## Output Columns

ID, TITLE, SEVERITY, PROGRESS, CHANNEL, CREATED.

## Examples

```bash
# Critical incidents in the last hour
flashduty incident list --severity Critical --since 1h

# Triggered incidents containing "database" in the title
flashduty incident list --progress Triggered --title "database"

# Closed incidents from a specific channel, page 2
flashduty incident list --progress Closed --channel 12345 --page 2
```
