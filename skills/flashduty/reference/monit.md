# fduty monit — command card

Prereq: `SKILL.md` read. Flashmonit is five separate surfaces sharing one command group, so this card is an index: **read the card for the surface you need, not all of them.**

## Route here when

"监控规则 / 告警规则 / 数据源 / PromQL查询 / 日志查询 / 诊断 / 监控目标 / 主机工具" or "alert rule / datasource / metric query / log pattern / diagnose / monitored host / tools catalog" → **monit**. NOT `incident` (that domain = the alert graph after rules fire), and **"数据源" here means a system Flashmonit queries** — On-call 集成 / 告警来源 is a different surface (`reference/channel.md`), and the top-level `datasource` group (`fduty datasource im-war-room-enabled-list`) is On-call IM plumbing, not this one.

## Which card

| surface | intent | card |
|---|---|---|
| Datasources | connect / list / inspect a datasource, SLS discovery | **`reference/monit-datasource.md`** |
| Alert rules | rule CRUD, folders, counters, audits, export/import | **`reference/monit-rule.md`** |
| Probing | ad-hoc query, log-pattern / metric-trend RCA, targets, on-box tools | **`reference/monit-probe.md`** |
| Service map | fleet, topology, status | **`reference/monit-servicemap.md`** |
| Store rulesets | ruleset CRUD | **`reference/monit-ruleset.md`** |

Key IDs are shared across all of them: **rule ID (int)** from `rule-list-basic`; **datasource name (string)** — never guess, always discover via `datasource-list` (see `reference/monit-datasource.md`).

Read verbs are free. Mutating verbs change state — confirm before running; each card flags its own, and marks the irreversible ones.

<!-- GENERATED:monit START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->


<!-- GENERATED:monit END -->
