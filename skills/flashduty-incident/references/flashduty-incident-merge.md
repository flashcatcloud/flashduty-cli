# flashduty incident merge

Merge source incidents into a target incident. **This operation is IRREVERSIBLE.**

## Usage

```bash
flashduty incident merge <target_id> --source <id1,id2,...>
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source` | string | | Comma-separated source incident IDs (**required**, max 100) |

## Examples

```bash
flashduty incident merge abc123 --source def456,ghi789,jkl012
```

## Notes

- This operation is **IRREVERSIBLE**. Always double-check source and target IDs before running.
- Source incidents are absorbed into the target; they cannot be un-merged.
