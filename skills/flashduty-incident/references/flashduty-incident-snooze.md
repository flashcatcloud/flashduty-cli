# flashduty incident snooze

Snooze one or more incidents for a specified duration.

## Usage

```bash
flashduty incident snooze <id> [<id2> ...] --duration <duration>
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--duration` | string | | Duration, e.g. `2h`, `30m` (**required**, max `24h`, must be whole minutes) |

## Examples

```bash
flashduty incident snooze abc123 --duration 2h
```

## Notes

- Maximum snooze duration is 24 hours.
- Duration must be specified in whole minutes (e.g., `30m`, `1h`, `2h`).
