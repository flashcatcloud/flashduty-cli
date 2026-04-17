# flashduty incident timeline

View the incident timeline/event history (non-paginated).

## Usage

```bash
flashduty incident timeline <id>
```

## Flags

No command-specific flags. Supports global flags (`--json`, `--no-trunc`, etc.).

## Output Columns

TIME, TYPE, OPERATOR, DETAIL.

## Examples

```bash
flashduty incident timeline abc123
```

## Notes

Unlike `incident feed`, this command returns the full timeline without pagination. Use `incident feed` for large timelines where you need paginated access.
