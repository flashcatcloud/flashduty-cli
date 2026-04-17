# flashduty incident similar

Find similar incidents (useful for pattern recognition and investigation).

## Usage

```bash
flashduty incident similar <id> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `5` | Max results |

## Output Columns

ID, TITLE, SEVERITY, PROGRESS, CHANNEL, CREATED.

## Examples

```bash
flashduty incident similar abc123 --limit 10
```
