---
name: flashduty
version: "3.0"
description: "USE FIRST for Flashduty tasks — status pages, incidents, alerts, on-call, monitors, automations, RUM, members. `fduty` CLI = the whole API. ALWAYS load this skill + read reference/<domain>.md for exact verbs & flags BEFORE running fduty. Don't guess or --help-dance."
allowed-tools: bash, read
hidden: true  # internal-only: withheld from skills.sh public discovery (Safari embeds this skill directly).
---

# Flashduty CLI

`fduty` is your interface to Flashduty — invoke it from `bash`. This SKILL.md is a **router**: shared model + conventions below, then a domain index. For a real task, **read the one `reference/<domain>.md` card first** — it carries every command, flag, enum, and the worked flow for that domain, so you operate without `--help` trial-and-error or guessing command names.

## Auth & availability

- **Auth.** Set your Flashduty app key once — `export FLASHDUTY_APP_KEY=<key>` — or pass `--app-key <key>` per call. Then just call the verb.
- **No curl for the API.** The CLI is the only supported path to Flashduty — never hand-roll an HTTP call.
- **If `fduty: command not found`** (rare — it is normally on PATH at startup): install from the Flashduty CDN into a user-writable dir (no sudo, no hang), then tell the user — don't work around it: `curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | FLASHDUTY_INSTALL_DIR="$HOME/.local/bin" INSTALLED_NAME=fduty sh && export PATH="$HOME/.local/bin:$PATH"`.

## Data model — 3 layers

`Alert Event` (raw signal) → `Alert` (deduplicated by `alert_key`) → `Incident` (actionable; 1–5000 alerts). `Change` events correlate to incidents by shared labels + time proximity (no foreign key). Command groups map to these layers: `alert-event`, `alert`, `incident`, `change`.

## Output — prefer `toon`

Append `--output-format toon` to read commands: it drops the per-row repeated keys that JSON emits, so lists cost far fewer tokens. Use `--json` only to pipe into `jq`. Bare output is a human table — don't parse it.

**Empty result = authoritative not-found.** A filter returning `[]` means no such entity in scope — report it (optionally the 1–2 closest names) and stop. Do **not** brute-force (no shifted-keyword re-queries, no widening past caps, no full-dump grep). Never infer "feature not enabled" from an empty list, and never fabricate data absent from tool output.

**A result you did not fetch is "unknown", never "empty".** You may report a command's result — including "returned empty" or any count/list/finding — **only if that exact command appears in your tool-call history this turn**. If you did not run it, the honest answer is "未查询 / not queried", followed by the command to run. Writing "`incident similar` 返回空" or "无变更" for a command you never executed is fabrication, not a summary.

## Command names — don't guess, read the card

The hot path: **read the domain card** (index below) for the exact verb + flags. Command groups are hyphenated (`status-page`, `alert-event`), not concatenated (`statuspage`) — guessing the wrong form costs a failed call. For a command outside the cards, derive it from its API path: **group = first path segment, verb = the rest joined by `-`** (`POST /status-page/change/create` → `fduty status-page change-create`), then confirm with `fduty <group> <verb> --help`. Pass nested-object / array fields as JSON via `--data '{...}'`; typed scalar flags override matching `--data` keys.

**Positional arguments.** A card heading like `### change-create <page-id>` means that id is **positional** — pass it as the first bare argument (`change-create 5759… --type incident`), not as `--page-id`. A heading with no `<…>` takes all inputs as flags. Cards mark each positional with a `(positional, required)` row; trust the heading over your instinct to use a flag.

## fduty answers directly — don't grep or browse

Configuration, permission-model, enrichment, monitor, and on-call questions are answered by `fduty` itself (the cards + the live commands). Do **not** grep external documentation or browse the web for something the CLI covers — that usually returns staler information than the live API. Read the card, run the verb.

## Safety — confirm before mutating

Read verbs (`list`, `get`, `info`, `detail`, `timeline`) are free. Mutating verbs (`create`, `update`, `delete`, `merge`, `ack`, `close`, `assign`, `move`, …) change state — recommend the action and get explicit per-target confirmation first. `merge` / `delete` are **irreversible** — double-check IDs. `create` notifies responders/subscribers. `list` before any bulk mutate to confirm the IDs.

## Compound flows — bundled scripts

Some asks span several commands. For those the skill ships a script that fetches everything in one call — run it as your **first action** for that ask, rather than hand-picking commands and writing the rest from memory:

- **Full incident fault analysis** (详情 + 关联告警 + 变更 + 时间线 + 相似故障 + 复盘 / detail + alerts + changes + timeline + similar + post-mortems): `bash scripts/incident-summary.sh <incident-id>` — runs all six reads and prints them in one block, so the summary is written from real output. See `reference/incident.md`.

## Domain index — read the card for the task

| intent / 意图 (terms route in either language) | card |
|---|---|
| incident / fault / 故障 / 事件 / triage 分诊 / acknowledge 认领 / merge 合并 / escalate 升级 / postmortem 复盘 / **summarize or analyze an incident 故障汇总分析** | **`reference/incident.md`** |
| alert / 告警 / dedup 去重 / alert fields 告警字段 / alert pipeline 告警管道 | **`reference/alert.md`** |
| change / 变更 / deployment 部署 / release 发布 / correlated change 变更关联 / what changed | **`reference/change.md`** |
| monitor / 监控 / alert rule 告警规则 / datasource 数据源 / inspection 巡检 / rule config 规则配置 | **`reference/monit.md`** |
| automation / 自动化 / 定时 AI SRE / scheduled AI task / daily brief / weekly report / webhook trigger / POST trigger / chat-created automation | **`reference/automation.md`** |
| metric/log query / 指标查询 / 日志查询 / PromQL / LogsQL / SQL / trend 趋势 / log clustering 日志聚类 / datasource RCA 数据源排查 | **`reference/monit-query.md`** |
| host diagnostics / 主机诊断 / on-box / process 进程 / load 负载 / lock 锁 / slow query 慢查询 / mysql / reachability 可达性 | **`reference/monit-agent.md`** |
| channel / 协作空间 / collaboration space / 频道 / integration 集成 / dispatch rule 分派规则 / escalation 升级规则 / noise reduction 降噪 / silence 静默 / inhibit 抑制 | **`reference/channel.md`** |
| enrichment / 数据加工 / 富化 / label mapping 字段映射 / extraction 提取 / mapping schema 集成 schema | **`reference/enrichment.md`** |
| insight / 洞察 / stats 统计 / trend 趋势 / MTTA / MTTR / top alerts Top 告警 / incident export 故障导出 | **`reference/insight.md`** |
| schedule / on-call / 值班 / 排班 / rotation 轮值 / who is on call 谁在值班 / shift 班次 / next responder 下一班 | **`reference/schedule.md`** |
| calendar / 日历 / on-call calendar 值班日历 / calendar event 日历事件 / holiday 休假 | **`reference/calendar.md`** |
| template / 通知模板 / message template 消息模板 / card template 卡片模板 | **`reference/template.md`** |
| role / 角色 / permission 权限 / RBAC / permission factor 权限因子 | **`reference/role.md`** |
| team / 团队 / team membership 团队成员归属 / HR sync HR 同步 | **`reference/team.md`** |
| member / 成员 / person 人员 / invite 邀请 / grant role 角色授予 | **`reference/member.md`** |
| custom field / 自定义字段 / field option 字段选项 | **`reference/field.md`** |
| route / 分派路由 / alert routing 告警路由 / integration routing 集成路由 / routing case 路由用例 | **`reference/route.md`** |
| RUM / real user monitoring / 真实用户监控 / frontend 前端 / application 应用 / issue | **`reference/rum.md`** |
| status page / 状态页 / public incident 公开事件 / public timeline 公开时间线 / maintenance window 维护窗口 / subscriber 订阅者 | **`reference/status-page.md`** |
