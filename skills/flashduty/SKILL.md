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
- **No curl for the API.** The CLI is the only supported path to Flashduty — never hand-roll an HTTP call.
- **If `fduty: command not found`** (rare — it is normally on PATH at startup): install from the Flashduty CDN into a user-writable dir (no sudo, no hang), then tell the user — don't work around it: `curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | FLASHDUTY_INSTALL_DIR="$HOME/.local/bin" INSTALLED_NAME=fduty sh && export PATH="$HOME/.local/bin:$PATH"`.

## Data model — 3 layers

`Alert Event` (raw signal) → `Alert` (deduplicated by `alert_key`) → `Incident` (actionable; 1–5000 alerts). `Change` events correlate to incidents by shared labels + time proximity (no foreign key). Command groups map to these layers: `alert-event`, `alert`, `incident`, `change`.

## Output — prefer `toon`

Append `--output-format toon` to read commands: it drops the per-row repeated keys that JSON emits, so lists cost far fewer tokens. Use `--json` only to pipe into `jq`. Bare output is a human table — don't parse it.

**Empty result = authoritative not-found.** A filter returning `[]` means no such entity in scope — report it (optionally the 1–2 closest names) and stop. Do **not** brute-force (no shifted-keyword re-queries, no widening past caps, no full-dump grep). Never infer "feature not enabled" from an empty list, and never fabricate data absent from tool output.

## Command names — don't guess, read the card

The hot path: **read the domain card** (index below) for the exact verb + flags. Command groups are hyphenated (`status-page`, `alert-event`), not concatenated (`statuspage`) — guessing the wrong form costs a failed call. For a command outside the cards, derive it from its API path: **group = first path segment, verb = the rest joined by `-`** (`POST /status-page/change/create` → `fduty status-page change-create`), then confirm with `fduty <group> <verb> --help`. Pass nested-object / array fields as JSON via `--data '{...}'`; typed scalar flags override matching `--data` keys.

**Positional arguments.** A card heading like `### change-create <page-id>` means that id is **positional** — pass it as the first bare argument (`change-create 5759… --type incident`), not as `--page-id`. A heading with no `<…>` takes all inputs as flags. Cards mark each positional with a `(positional, required)` row; trust the heading over your instinct to use a flag.

## fduty answers directly — don't dispatch or grep

Configuration, permission-model, enrichment, monitor, and on-call questions are answered by `fduty` itself (the cards + the live commands). Do **not** grep documentation, browse the web, or dispatch a subagent / product-guide agent for something the CLI covers — that wastes a turn and usually returns staler information than the live API. Read the card, run the verb.

## Safety — confirm before mutating

Read verbs (`list`, `get`, `info`, `detail`, `timeline`) are free. Mutating verbs (`create`, `update`, `delete`, `merge`, `ack`, `close`, `assign`, `move`, …) change state — recommend the action and get explicit per-target confirmation first. `merge` / `delete` are **irreversible** — double-check IDs. `create` notifies responders/subscribers. `list` before any bulk mutate to confirm the IDs.

## Domain index — read the card for the task

| 意图 / intent | card |
|---|---|
| 事件 / 故障 / incident / 分诊 / 认领 / 合并 / 升级 / 事后复盘 / 告警关联 | **`reference/incident.md`** |
| 告警 / alert / 去重 / 告警字段 / 告警管道 / 告警关联事件 | **`reference/alert.md`** |
| 监控 / monitor / 告警规则 / 数据源 / 监控诊断 / 巡检 / 规则配置 | **`reference/monit.md`** |
| 指标查询 / 日志查询 / PromQL / LogsQL / SQL 验证 / 趋势 / 日志聚类 / 数据源 RCA | **`reference/monit-query.md`** |
| 主机诊断 / on-box / 进程 / 负载 / 锁 / 慢查询 / mysql 诊断 / 可达性 | **`reference/monit-agent.md`** |
| 协作空间 / channel / 集成 / 分派规则 / 升级规则 / 降噪 | **`reference/channel.md`** |
| 数据加工 / 富化 / enrichment / 集成 schema / 字段映射 / 提取 | **`reference/enrichment.md`** |
| 洞察 / 统计 / 趋势 / MTTA / MTTR / Top 告警 / 故障导出 | **`reference/insight.md`** |
| 谁在值班 / 当前值班 / on-call / 值班人查询 | **`reference/oncall.md`** |
| 排班 / 轮值 / schedule / 值班层级 / 排班预览 | **`reference/schedule.md`** |
| 日历 / calendar / 值班日历 / 日历事件 / 休假日历 | **`reference/calendar.md`** |
| 通知模板 / 消息模板 / template / 卡片模板 | **`reference/template.md`** |
| 角色 / 权限 / role / RBAC / 权限因子 | **`reference/role.md`** |
| 团队 / team / 团队成员归属 / HR 同步 | **`reference/team.md`** |
| 成员 / 人员 / member / 邀请 / 角色授予 | **`reference/member.md`** |
| 自定义字段 / custom field / 字段选项 | **`reference/field.md`** |
| 分派路由 / route / 集成路由 / 路由用例 | **`reference/route.md`** |
| 真实用户监控 / RUM / 前端 / 应用 / issue | **`reference/rum.md`** |
| 公开事件 / 状态页 / 公开时间线 / 维护窗口 / 订阅者 / 状态页迁移 | **`reference/status-page.md`** |
