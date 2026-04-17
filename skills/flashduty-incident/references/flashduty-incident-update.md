# flashduty incident update

Update an existing incident's fields.

## Usage

```bash
flashduty incident update <id> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--title` | string | | New title |
| `--description` | string | | New description |
| `--severity` | string | | New severity: `Critical`, `Warning`, `Info` |
| `--field` | string array | | Custom field `key=value` (repeatable) |

## Examples

```bash
# Update severity and add custom fields
flashduty incident update abc123 --severity Warning --field "team=platform" --field "region=us-east-1"

# Update title
flashduty incident update abc123 --title "Resolved: Payment gateway timeout"
```
