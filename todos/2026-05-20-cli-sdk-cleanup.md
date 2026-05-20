# TODO: Remove CLI Raw API Client

## Goal

Update `flashduty-cli` to consume SDK methods for incident lifecycle and war-room commands.

## Tasks

- [x] Replace CLI-local incident lifecycle client methods with calls to `flashduty-sdk`.
- [x] Remove duplicate CLI-local request/response types where SDK types can be used directly.
- [x] Remove the CLI raw HTTP client after SDK migration.
- [x] Change `incident war-room create` so `--integration` is optional.
- [x] When `--integration` is omitted, call SDK datasource discovery for enabled war-room IM integrations and use the first returned `data_source_id` as `integration_id`.
- [x] Keep `--integration` as an override for explicit selection.
- [x] Add focused CLI command tests for auto-discovery and explicit override behavior.
- [x] Replace the temporary local SDK module replacement with a real SDK pseudo-version after the SDK branch is pushed.
- [x] Run only task-relevant SDK and CLI tests before publishing.
