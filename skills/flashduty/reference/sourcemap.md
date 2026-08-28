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
- `--build-id` string — Android only. Filter by Gradle plugin build identifier. Max 200 characters. (≤200 chars)
- `--end-time` int64 (required) — End of upload time range, Unix epoch milliseconds. Maximum window: 365 days.
- `--kind` string — Symbol type filter, Android and HarmonyOS only (ignored for other platforms): 'mapping' (default) lists ProGuard/R8 mappings or ArkTS sourcemaps, 'native' lists native .so symbols. · enum: mapping | native
- `--limit` int64 — Page size. Maximum 100. Default 20. (max 100)
- `--orderby` string — Sort field; defaults to 'created_at' descending when omitted. · enum: created_at | updated_at
- `--page` int64 — Page number, starting at 1. (min 1)
- `--query` string — Free-text substring match. Matches 'minified_url' for the JS stores (browser/react-native/harmony/miniprogram), 'build_id' for android/flutter/electron and harmony with 'kind=native', or 'uuid' for ios (case-insensitive, hyphens ignored). (≤200 chars)
- `--search-after-ctx` string
- `--services` stringSlice — Filter by service names. Up to 100 values.
- `--start-time` int64 (required) — Start of upload time range, Unix epoch milliseconds. Must be > 0 and before 'end_time'.
- `--type` string — Platform whose symbol store to list. Defaults to 'browser' when omitted; any other value returns an empty list. | Value | Store listed | |---|---| | 'browser' | JavaScript sourcemaps (shared store; excludes HarmonyOS ArkTS and React Native rows) | | 'android' | ProGuard/R8 mapping files; with 'kind=native', Android NDK .so symbols | | 'ios' | iOS dSYM symbol files | | 'miniprogram' | WeChat mini program sourcemaps | | 'react-native' | React Native JS sourcemaps | | 'harmony' | HarmonyOS ArkTS sourcemaps; with 'kind=native', HarmonyOS .so symbols | | 'flutter' | Flutter Dart AOT symbols | | 'electron' | Electron Breakpad symbols | · enum: browser | android | ios | miniprogram | react-native | harmony | flutter | electron
- `--uuid` string — iOS only. Filter by dSYM bundle UUID. Max 200 characters. (≤200 chars)
- `--versions` stringSlice — Filter by version strings. Up to 100 values.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (string); git_commit_sha (string); git_repository_url (string); key (string); metadata (object); minified_path (string); minified_url (string); service (string); size (integer); sourcemap_path (string); type (string); updated_at (string); version (string)

### stack-enrich
Enrich a stack trace
- `--arch` string — Android NDK architecture such as 'arm', 'arm64', 'x86', or 'x64'.
- `--build-id` string — Android build ID for Gradle plugin 1.13.0 and later.
- `--near` int64 — Number of nearby meaningful source lines to return around converted frames. (1-20)
- `--no-cache` bool — Skip cached enrich results. Intended for debugging.
- `--platform` string — Narrows a 'react-native' enrich to the app's native platform: 'ios' for the iOS native layer, 'android' for the Android native layer (the console derives it from the event's OS). Ignored for other 'type' values. · enum: ios | android
- `--service` string (required) — Application or service name used when the sourcemap was uploaded.
- `--source-type` string — Android error source type. Use 'ndk' with 'arch' for native symbolication.
- `--stack` string — Raw stack trace to parse and enrich.
- `--type` string — Source platform whose symbol store is used. Defaults to 'browser' when omitted. | Value | Symbolication | |---|---| | 'browser' | JavaScript stacks via sourcemaps | | 'android' | Java/Kotlin stacks via ProGuard/R8 mappings; native stacks via NDK symbols (send 'source_type=ndk' with 'arch') | | 'ios' | iOS crash stacks via dSYM (send 'binary_images') | | 'miniprogram' | WeChat mini program stacks via sourcemaps | | 'harmony' | HarmonyOS stacks via ArkTS sourcemaps or native symbols | | 'flutter' | Flutter/Dart stacks via Dart AOT symbols | | 'electron' | Electron JavaScript stacks via sourcemaps; minidump native frames via Breakpad symbols (derived from 'source_type') | | 'react-native' | React Native JS stacks via sourcemaps; narrow the lookup with 'platform' | · enum: browser | android | ios | miniprogram | harmony | flutter | electron | react-native
- `--variant` string — Android build variant used by older Gradle plugin versions.
- `--version` string (required) — Application version used when the sourcemap was uploaded.
- body-only (`--data`): binary_images (array<object>)
- response: single object (`data` unwrapped to the top level) — fields: frames (array<object>)

<!-- GENERATED:sourcemap END -->

## Gotchas

- **Top-level group:** use `fduty sourcemap ...`, not `fduty rum sourcemap ...`.
- **`stack-enrich` needs exact upload identity:** `type`, `service`, and `version` must match the uploaded sourcemap/dSYM metadata.
- **Use `--data` for stack traces.** Multiline stacks are easier and safer as JSON body payloads than shell-escaped flags.
- **Empty `list` is authoritative** for the supplied filters; re-check service/version/type from the RUM app or build metadata before changing the time window.
