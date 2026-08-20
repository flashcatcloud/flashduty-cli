# fduty monit — service map

Prereq: `SKILL.md` + `reference/monit.md` read. The service map is Flashmonit's view of what is deployed and how it connects: the agent fleet, the topology between services, and their current status.

## Route here when

"服务地图 / 拓扑 / 服务依赖 / 探针队列" or "service map / topology / service dependencies / agent fleet" → this card. For diagnosing one specific target rather than surveying the fleet, use `reference/monit-probe.md`.

All five verbs are read-only.

<!-- GENERATED:monit[servicemap] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### servicemap-fleet
Browse service map fleet hosts
- `--agent-versions` stringSlice — Filter to hosts on any of these exact agent versions. Up to 20 values.
- `--capture-modes` stringSlice — Filter to hosts using any of these capture modes. 'unknown' matches hosts that have not reported a capture mode yet. · enum: ebpf | polling | unknown
- `--cursor` string — Opaque pagination cursor. Pass back the exact value from a previous response's 'next_cursor'; omit for the first page.
- `--edge-clusters` stringSlice — Filter to hosts in any of these exact edge cluster names. Up to 20 values.
- `--limit` int64 — Maximum number of matching hosts to return in this page. Default 50, range 1-100. (1-100)
- `--scan-limit` int64 — Maximum number of candidate hosts to examine while filling this page. Default 1000, range 'limit'-2000. (max 2000)
- `--statuses` stringSlice — Filter to hosts currently in any of these statuses. Up to 20 values. · enum: active | degraded | stale | initializing | disabled | unsupported | no_data
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); generated_at_ms (string); items (array<object>); next_cursor (string); partial (boolean); truncated (boolean); truncation_reasons (array<string>)

### servicemap-fleet-summary
Get service map fleet summary
- `--agent-versions` stringSlice — Filter to hosts on any of these exact agent versions. Up to 20 values.
- `--capture-modes` stringSlice — Filter to hosts using any of these capture modes. 'unknown' matches hosts that have not reported a capture mode yet. · enum: ebpf | polling | unknown
- `--edge-clusters` stringSlice — Filter to hosts in any of these exact edge cluster names. Up to 20 values.
- `--scan-limit` int64 — Maximum number of candidate hosts to scan. Default 2000, range 1-5000. (1-5000)
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); generated_at_ms (string); partial (boolean); scan_limit (integer); truncated (boolean); truncation_reasons (array<string>)

### servicemap-status
Get service map status
- `--fleet` bool — When 'true', ignore 'host_id'/'host_ids' and instead sample up to 'limit' fleet candidate hosts for the account. Default 'false'.
- `--host-id` string — A single host ID to check. Combine with 'host_ids' to check several; mutually exclusive with 'fleet=true'. (≤128 chars)
- `--host-ids` stringSlice — Multiple host IDs to check in one call, up to 200 combined with 'host_id'. Mutually exclusive with 'fleet=true'.
- `--limit` int64 — In 'fleet' mode, the number of candidate hosts to sample. Ignored otherwise. Default 100, range 1-200. (1-200)
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); fleet (boolean); generated_at_ms (string); items (array<object>); partial (boolean)

### servicemap-summary
Get service map summary
- `--network-scope-id` string — Optional integrity check: if set, must match the network scope already associated with 'anchor.host_id', or the request is rejected with 'InvalidParameter'.
- body-only (`--data`): anchor (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: anchor_entity_id (string); anchor_host_id (string); authoritative (boolean); context_ref_detail (string); coverage (object); freshness (object); graph_role (string); latest_collection_authoritative (boolean); latest_health_at_ms (string); neighbors (array<object>); network_scope_id (string); observed_at_ms (string); received_at_ms (string); resolution_counts (object); status (string); truncated (boolean); truncation_reasons (array<string>)

### servicemap-topology
Get service map topology
- `--at` string — Time selector for the query. Only 'now' is currently supported; omitting the field behaves the same. · enum: now
- `--depth` int64 — Maximum traversal depth from the anchor. Default 1, maximum 3. (max 3)
- `--direction` string — Traversal direction. Only 'outbound' is currently supported; omitting the field behaves the same. · enum: outbound
- `--include-metrics` bool — Whether to include the raw per-edge 'metrics' payload in the response. Default 'false'.
- `--max-edges` int64 — Maximum number of edges to examine before truncating. Default 200, maximum 1000. (max 1000)
- `--max-nodes` int64 — Maximum number of nodes to return before truncating. Default 100, maximum 500. (max 500)
- `--network-scope-id` string — Optional integrity check: if set, must match the network scope already associated with 'anchor.host_id', or the request is rejected with 'InvalidParameter'.
- `--unresolved-mode` string — How unresolved edges are projected. 'full' (default) includes them in 'edges' and 'unresolved_endpoints'; 'summary' omits them from 'edges' and returns only a bounded sample in 'unresolved_endpoints'. · enum: summary | full
- body-only (`--data`): anchor (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: anchor_entity_id (string); anchor_host_id (string); coverage (object); edges (array<object>); freshness (object); network_scope_id (string); nodes (array<object>); observed_at_ms (string); resolution_counts (object); truncated (boolean); truncation_reasons (array<string>); unresolved_endpoints (array<object>); unresolved_projection (object)

<!-- GENERATED:monit[servicemap] END -->
