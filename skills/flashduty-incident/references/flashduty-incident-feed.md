# flashduty incident feed

Paginated timeline of incident events.

## Usage

```bash
flashduty incident feed <id> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `20` | Max events |
| `--page` | int | `1` | Page number |

## Output Columns

TIME, TYPE, OPERATOR, DETAIL.

## Examples

```bash
flashduty incident feed abc123 --limit 50 --page 2
```
