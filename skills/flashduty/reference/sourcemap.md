# fduty sourcemap — command card

Prereq: `SKILL.md` read. These verbs are read-only/debugging helpers. They do not upload, delete, or mutate sourcemap records.

## Route here when

"sourcemap / source map / source mapping / 代码映射 / 堆栈还原 / symbolication / deobfuscate / stack trace / stack enrich / dSYM / miniprogram source map" → **sourcemap**, NOT `rum sourcemap`. RUM data queries live under `reference/rum.md`; uploaded mapping-file lookup and stack enrichment live here.

## Intent → verb

| want | verb |
|---|---|
| list uploaded sourcemap or dSYM records | `list` |
| enrich / deobfuscate a minified stack trace | `stack-enrich` |

## Hot flow — enrich a browser stack trace

```bash
# 1. Confirm the service/version/type actually has uploaded mapping files
fduty sourcemap list \
  --type browser \
  --services checkout-web \
  --start-time 1712000000000 \
  --end-time 1712700000000 \
  --output-format toon

# 2. Enrich the minified stack trace. Use --data for multiline stack payloads.
fduty sourcemap stack-enrich \
  --data '{"type":"browser","service":"checkout-web","version":"1.0.0","near":3,"stack":"TypeError: Cannot read properties of undefined\n    at render (https://cdn.example.com/app.min.js:1:2345)"}' \
  --output-format toon
```

<!-- GENERATED:sourcemap START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### list
List sourcemaps
- `--asc` bool — Sort ascending. Default false (descending).
- `--build-id` string — Android only. Filter by Gradle plugin build identifier. Max 200 characters.
- `--end-time` int64 (required) — End of upload time range, Unix epoch milliseconds. Maximum window: 365 days.
- `--limit` int64 — Page size. Maximum 100. Default 20. (max 100)
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number, starting at 1. (min 1)
- `--query` string — Substring match on the minified URL (browser) or build ID (android). Max 200 characters.
- `--search-after-ctx` string
- `--services` stringSlice — Filter by service names. Up to 100 values.
- `--start-time` int64 (required) — Start of upload time range, Unix epoch milliseconds. Must be > 0 and before 'end_time'.
- `--type` string — Platform type. Defaults to 'browser' when omitted. · enum: browser | android | ios
- `--uuid` string — iOS only. Filter by dSYM bundle UUID. Max 200 characters.
- `--versions` stringSlice — Filter by version strings. Up to 100 values.

### stack-enrich
Enrich a stack trace
- `--arch` string — Android NDK architecture such as 'arm', 'arm64', 'x86', or 'x64'.
- `--build-id` string — Android build ID for Gradle plugin 1.13.0 and later.
- `--near` int64 — Number of nearby meaningful source lines to return around converted frames. (1-20)
- `--no-cache` bool — Skip cached enrich results. Intended for debugging.
- `--service` string (required) — Application or service name used when the sourcemap was uploaded.
- `--source-type` string — Android error source type. Use 'ndk' with 'arch' for native symbolication.
- `--stack` string — Raw stack trace to parse and enrich.
- `--type` string — Source platform. Defaults to 'browser' when omitted. · enum: browser | android | ios | miniprogram | harmony
- `--variant` string — Android build variant used by older Gradle plugin versions.
- `--version` string (required) — Application version used when the sourcemap was uploaded.
- body-only (`--data`): binary_images (array<object>)

<!-- GENERATED:sourcemap END -->

## Gotchas

- **Top-level group:** use `fduty sourcemap ...`, not `fduty rum sourcemap ...`.
- **`stack-enrich` needs exact upload identity:** `type`, `service`, and `version` must match the uploaded sourcemap/dSYM metadata.
- **Use `--data` for stack traces.** Multiline stacks are easier and safer as JSON body payloads than shell-escaped flags.
- **Empty `list` is authoritative** for the supplied filters; re-check service/version/type from the RUM app or build metadata before changing the time window.
