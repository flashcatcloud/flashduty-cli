# flashduty incident reassign

Reassign an incident to new responders.

## Usage

```bash
flashduty incident reassign <id> --person <id1,id2,...>
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--person` | string | | Comma-separated person IDs (**required**) |

## Examples

```bash
flashduty incident reassign abc123 --person 101,102
```

## Notes

Use `flashduty member list` to find person IDs.
