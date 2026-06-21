# fduty monit-agent — command card

Prereq: `SKILL.md` read. On-box diagnostics: run diagnostic tools on a host or database target via its installed monit-agent. Both verbs are read-only probes. Pairs with **`monit-query`** (datasource-side RCA).

## Route here when

"主机诊断 / 进程 / 负载 / 锁 / 慢查询 / mysql 诊断 / 可达性 / on-box / 看那台机器上发生了什么" → **monit-agent**. You need a **target locator** (host/instance identifier). Always `catalog` first to learn what tools that target exposes — tool names are not guessable.

## Intent → verb

| want | verb |
|---|---|
| list the diagnostic tools available for a target | `catalog --target-locator <t>` |
| run up to 8 of those tools on the target | `invoke --target-locator <t> --data '{"tools":[…]}'` |

## Hot flow — diagnose a host

```bash
# 1. see which tools this target exposes (tool names come from here, never guess)
fduty monit-agent catalog --target-locator <host-or-instance> --output-format toon
# 2. invoke up to 8 tools concurrently; tool names taken verbatim from the catalog
fduty monit-agent invoke --target-locator <host-or-instance> \
  --data '{"tools":[{"tool":"host.top","params":{}},{"tool":"host.disk","params":{}}]}'
```

<!-- GENERATED:monit-agent START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### catalog
List the diagnostic tools the agent exposes for a target
- `--target-kind` string
- `--target-locator` string

### invoke
Run up to 8 monit-agent tools concurrently on a target
- `--target-kind` string
- `--target-locator` string

<!-- GENERATED:monit-agent END -->

## Key concepts

- **`catalog` → `invoke` is the order.** `catalog` returns each tool's `name` (+ `input_schema` for its params); `invoke` runs them. Tool names are target-specific — take them verbatim from the catalog, do not invent.
- **`invoke` carries the tool list in `--data`**: `{"tools":[{"tool":"<name>","params":{…}}, … up to 8]}`. `params` defaults to `{}`. `--target-locator` (required) and `--target-kind` override matching `--data` keys.
- Each result carries `agent_elapsed_ms` (agent-side) vs `e2e_elapsed_ms` (end-to-end) — a large gap signals network/edge slowness, not a slow tool.

## Gotchas

- **Quoted/comma params (e.g. SQL) → use `--data -` with a heredoc** to avoid shell-quoting hell:
  ```bash
  fduty monit-agent invoke --target-locator 'db-1' --data - <<'FDUTY'
  {"tools":[{"tool":"mysql.query","params":{"sql":"SELECT a, b FROM t WHERE s='RUNNING'","max_rows":50}}]}
  FDUTY
  ```
- **`ambiguous_target_kind` error** ⇒ the locator matched multiple kinds; re-issue with `--target-kind`.
- A `target_unavailable` / `target_unreachable` error means the agent isn't connected — report it; don't retry endlessly or fall back to SSH.
- Per-tool errors (`timeout`, `denied`, `unknown_tool`…) are reported per result, mutually exclusive with that tool's `data`.

## Worked example — top processes + disk on a host

```bash
fduty monit-agent invoke --target-locator web-prod-3 \
  --data '{"tools":[{"tool":"host.top","params":{"limit":10}},{"tool":"host.disk","params":{}}]}' \
  --output-format toon
```
