# flashduty incident create

Create a new incident. In interactive terminals, title and severity prompt interactively if not provided via flags.

## Usage

```bash
flashduty incident create [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--title` | string | | Incident title (**required**, 3-200 chars) |
| `--severity` | string | | `Critical`, `Warning`, or `Info` (**required**) |
| `--channel` | int | | Channel ID |
| `--description` | string | | Description (max 6144 chars) |
| `--assign` | int slice | | Person IDs to assign (repeatable flag) |

## Examples

```bash
# Create with all flags
flashduty incident create --title "Payment gateway timeout" --severity Critical --channel 100 --description "Stripe API returning 504s" --assign 1 --assign 2

# Minimal - will prompt interactively for title and severity in a terminal
flashduty incident create
```

## Notes

- Creating an incident triggers notifications to responders -- confirm with the user before running.
- In non-interactive environments (pipes, scripts), `--title` and `--severity` must be provided via flags.
