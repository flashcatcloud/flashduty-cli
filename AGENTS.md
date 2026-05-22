# Agent Instructions

## Flashduty SDK Boundary

- Do not implement Flashduty public API endpoint clients directly in this CLI repository.
- If a CLI command needs an endpoint that is missing from `github.com/flashcatcloud/flashduty-sdk`, add the typed adapter to `flashduty-sdk` first, with focused SDK tests.
- The CLI should consume SDK methods and keep only command parsing, output formatting, and CLI-specific orchestration.
- Existing raw HTTP adapters in the CLI are migration debt. Prefer removing them as SDK coverage catches up.
