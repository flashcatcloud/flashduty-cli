---
name: flashduty
version: "3.0"
description: "USE FIRST for Flashduty tasks — status pages, incidents, alerts, on-call, monitors, RUM, members. `fduty` CLI = the whole API. ALWAYS load this skill + read reference/<domain>.md for exact verbs & flags BEFORE running fduty. Don't guess or --help-dance."
allowed-tools: bash, read, task
---

# Flashduty CLI

`fduty` is your interface to Flashduty — invoke it from `bash`. This SKILL.md is a **router**: shared model + conventions below, then a domain index. For a real task, **read the one `reference/<domain>.md` card first** — it carries every command, flag, enum, and the worked flow for that domain, so you operate without `--help` trial-and-error or guessing command names.

## Auth & availability

- **Auth is automatic.** Safari injects the credential into each `fduty` subprocess. You never handle it and *cannot* read it — commands that reference `FLASHDUTY_APP_KEY` / `$FLASHDUTY_*` by name or dump the environment are rejected. Just call the verb.
- **No curl.** The CLI is the only supported path.

## Data model — 3 layers

`Alert Event` (raw signal) → `Alert` (deduplicated by `alert_key`) → `Incident` (actionable; 1–5000 alerts). `Change` events correlate to incidents by shared labels + time proximity (no foreign key). Command groups map to these layers: `alert-event`, `alert`, `incident`, `change`.

## Output — prefer `toon`

Append `--output-format toon` to read commands: it drops the per-row repeated keys that JSON emits, so lists cost far fewer tokens. Use `--json` only to pipe into `jq`. Bare output is a human table — don't parse it.

**Empty result = authoritative not-found.** A filter returning `[]` means no such entity in scope — report it (optionally the 1–2 closest names) and stop. Do **not** brute-force (no shifted-keyword re-queries, no widening past caps, no full-dump grep). Never infer "feature not enabled" from an empty list, and never fabricate data absent from tool output.

## Command names — don't guess, read the card

The hot path: **read the domain card** (index below) for the exact verb + flags. Command groups are hyphenated (`status-page`, `alert-event`), not concatenated (`statuspage`) — guessing the wrong form costs a failed call. For a command outside the cards, derive it from its API path: **group = first path segment, verb = the rest joined by `-`** (`POST /status-page/change/create` → `fduty status-page change-create`), then confirm with `fduty <group> <verb> --help`. Pass nested-object / array fields as JSON via `--data '{...}'`; typed scalar flags override matching `--data` keys.

## Safety — confirm before mutating

Read verbs (`list`, `get`, `info`, `detail`, `timeline`) are free. Mutating verbs (`create`, `update`, `delete`, `merge`, `ack`, `close`, `assign`, `move`, …) change state — recommend the action and get explicit per-target confirmation first. `merge` / `delete` are **irreversible** — double-check IDs. `create` notifies responders/subscribers. `list` before any bulk mutate to confirm the IDs.

## Domain index — read the card for the task

| 意图 / intent | card |
|---|---|
| 公开事件 / 状态页 / 公开时间线 / 维护窗口 / 订阅者 / 状态页迁移 | **`reference/status-page.md`** |

<!-- ROLLOUT: incident / alert / change / oncall / monit / rum / insight / channel / member / team / role / enrichment / route / template / schedule / … cards added after the status-page PoC proves the metrics. -->
