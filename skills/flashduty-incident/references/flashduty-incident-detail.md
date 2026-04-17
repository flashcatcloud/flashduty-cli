# flashduty incident detail

Full incident detail with AI-powered analysis fields: AI summary, root cause, resolution, and impact analysis.

## Usage

```bash
flashduty incident detail <id>
```

## Flags

No command-specific flags. Supports global flags (`--json`, `--no-trunc`, etc.).

## Output Fields

ID, Title, Severity, Progress, Channel, Created, Acknowledged, Closed, Alerts count, Events count, Frequency, AI Summary, Root Cause, Resolution, Impact, Description, Labels, Custom Fields, Responders.

## Examples

```bash
flashduty incident detail abc123
```

## Notes

This command provides richer output than `incident get` by including AI-generated analysis fields. Use this for deep investigation; use `incident get` for quick lookups.
