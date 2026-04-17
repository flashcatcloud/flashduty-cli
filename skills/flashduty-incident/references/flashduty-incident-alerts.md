# flashduty incident alerts

View alerts contributing to an incident.

## Usage

```bash
flashduty incident alerts <id> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `10` | Max alerts to show |

## Output Columns

ALERT_ID, TITLE, SEVERITY, STATUS, STARTED.

## Examples

```bash
flashduty incident alerts abc123 --limit 20
```
