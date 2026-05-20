# Flashduty CLI OpenAPI command gap

Updated: 2026-05-19

Sources:

- `https://docs.flashcat.cloud/api-reference/on-call.openapi.zh.json`
- `https://docs.flashcat.cloud/api-reference/platform.openapi.zh.json`
- `https://docs.flashcat.cloud/api-reference/monitors.openapi.zh.json`
- `https://docs.flashcat.cloud/api-reference/rum.openapi.zh.json`

This document tracks official OpenAPI endpoints that are not yet fully exposed
as `flashduty` commands. It intentionally focuses on command coverage, not
internal SDK helper coverage.

Legend:

- `missing`: no first-class CLI command.
- `partial`: some behavior exists, but the CLI does not expose the full API
  contract.
- `covered indirectly`: the CLI reaches the endpoint through another command or
  SDK enrichment path, but there is no user-facing command for the endpoint.
- `implemented`: implemented after this tracking document was introduced.
- `defer`: command shape or side effects need a separate design pass.

## Implemented incident lifecycle slice

These endpoints were selected as the first implementation slice because they
are close to the existing `incident` command family and have simple
request/response contracts.

| Endpoint | Status | Proposed command | Notes |
| --- | --- | --- | --- |
| `/incident/unack` | implemented | `incident unack <id> [<id2> ...]` | Inverse of `incident ack`; payload is `incident_ids`. |
| `/incident/wake` | implemented | `incident wake <id> [<id2> ...]` | Inverse of `incident snooze`; payload is `incident_ids`. |
| `/incident/comment` | implemented | `incident comment <id> [<id2> ...] --comment <text> [--mute-reply]` | Adds timeline comments. |
| `/incident/responder/add` | implemented | `incident add-responder <id> --person <ids>` | Additive responder change; distinct from replacement-style assignment. |
| `/incident/remove` | implemented | `incident remove <id> [<id2> ...] --force` | Destructive; requires confirmation outside JSON mode. |
| `/incident/disable-merge` | implemented | `incident disable-merge <id> [<id2> ...]` | Present in current docs.flashcat.cloud OpenAPI. |
| `/incident/assign` | partial | keep `incident reassign`; consider richer `incident assign` later | Existing command covers direct person assignment only. |
| `/incident/field/reset` | covered indirectly | `incident update --field key=value` | Already called once per custom field. |

Defer from the first slice:

| Endpoint | Status | Reason |
| --- | --- | --- |
| `/incident/custom-action/do` | defer | Integration-specific side effect; needs output and failure semantics. |
| `/incident/war-room/create` | implemented | Implemented as `incident war-room create`; supports `--add-observers`. |
| `/incident/war-room/delete` | implemented | Implemented as `incident war-room delete --force`. |
| `/incident/war-room/list` | implemented | Implemented as `incident war-room list`. |
| `/incident/war-room/detail` | implemented | Implemented as `incident war-room get`. |
| `/incident/post-mortem/info` | missing | Fits postmortem slice, not lifecycle. |
| `/incident/post-mortem/delete` | missing | Destructive; fits postmortem slice. |

## On-call gaps

### Alert management

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/alert/list-by-ids` | partial | `alert get` already supports detail by ID, but no batch list-by-ids command. |
| `/alert/pipeline/list` | missing | `alert pipeline list` |
| `/alert/pipeline/info` | missing | `alert pipeline get` |
| `/alert/pipeline/upsert` | missing | `alert pipeline upsert --file` |

### Calendar management

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/calendar/list` | missing | `calendar list` |
| `/calendar/info` | missing | `calendar get` |
| `/calendar/create` | missing | `calendar create --file` |
| `/calendar/update` | missing | `calendar update --file` |
| `/calendar/delete` | missing | `calendar delete --id --force` |
| `/calendar/event/list` | missing | `calendar event list` |
| `/calendar/event/upsert` | missing | `calendar event upsert --file` |
| `/calendar/event/delete` | missing | `calendar event delete --id --force` |

### Collaboration spaces

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/channel/info` | missing | `channel get` |
| `/channel/infos` | covered indirectly | Used for enrichment; no command. |
| `/channel/create` | missing | `channel create --file` |
| `/channel/update` | missing | `channel update --file` |
| `/channel/delete` | missing | `channel delete --id --force` |
| `/channel/enable` | missing | `channel enable --id` |
| `/channel/disable` | missing | `channel disable --id` |
| `/channel/escalate/rule/info` | missing | `escalation-rule get` |
| `/channel/escalate/rule/create` | missing | `escalation-rule create --file` |
| `/channel/escalate/rule/update` | missing | `escalation-rule update --file` |
| `/channel/escalate/rule/delete` | missing | `escalation-rule delete --id --force` |
| `/channel/escalate/rule/enable` | missing | `escalation-rule enable --id` |
| `/channel/escalate/rule/disable` | missing | `escalation-rule disable --id` |
| `/channel/notify/rule/list` | missing | `channel notify-rule list` |
| `/channel/notify/rule/create` | missing | `channel notify-rule create --file` |
| `/channel/notify/rule/update` | missing | `channel notify-rule update --file` |
| `/channel/notify/rule/delete` | missing | `channel notify-rule delete --id --force` |
| `/channel/notify/rule/enable` | missing | `channel notify-rule enable --id` |
| `/channel/notify/rule/disable` | missing | `channel notify-rule disable --id` |
| `/channel/silence/rule/list` | missing | `channel silence-rule list` |
| `/channel/silence/rule/create` | missing | `channel silence-rule create --file` |
| `/channel/silence/rule/update` | missing | `channel silence-rule update --file` |
| `/channel/silence/rule/delete` | missing | `channel silence-rule delete --id --force` |
| `/channel/silence/rule/enable` | missing | `channel silence-rule enable --id` |
| `/channel/silence/rule/disable` | missing | `channel silence-rule disable --id` |
| `/channel/inhibit/rule/list` | missing | `channel inhibit-rule list` |
| `/channel/inhibit/rule/create` | missing | `channel inhibit-rule create --file` |
| `/channel/inhibit/rule/update` | missing | `channel inhibit-rule update --file` |
| `/channel/inhibit/rule/delete` | missing | `channel inhibit-rule delete --id --force` |
| `/channel/inhibit/rule/enable` | missing | `channel inhibit-rule enable --id` |
| `/channel/inhibit/rule/disable` | missing | `channel inhibit-rule disable --id` |
| `/channel/unsubscribe/rule/list` | missing | `channel unsubscribe-rule list` |
| `/channel/unsubscribe/rule/create` | missing | `channel unsubscribe-rule create --file` |
| `/channel/unsubscribe/rule/update` | missing | `channel unsubscribe-rule update --file` |
| `/channel/unsubscribe/rule/delete` | missing | `channel unsubscribe-rule delete --id --force` |
| `/channel/unsubscribe/rule/enable` | missing | `channel unsubscribe-rule enable --id` |
| `/channel/unsubscribe/rule/disable` | missing | `channel unsubscribe-rule disable --id` |

### Label enrichment

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/enrichment/list` | missing | `enrichment list` |
| `/enrichment/info` | missing | `enrichment get` |
| `/enrichment/upsert` | missing | `enrichment upsert --file` |
| `/enrichment/mapping/api/list` | missing | `enrichment mapping-api list` |
| `/enrichment/mapping/api/info` | missing | `enrichment mapping-api get` |
| `/enrichment/mapping/api/create` | missing | `enrichment mapping-api create --file` |
| `/enrichment/mapping/api/update` | missing | `enrichment mapping-api update --file` |
| `/enrichment/mapping/api/delete` | missing | `enrichment mapping-api delete --id --force` |
| `/enrichment/mapping/schema/list` | missing | `enrichment mapping-schema list` |
| `/enrichment/mapping/schema/info` | missing | `enrichment mapping-schema get` |
| `/enrichment/mapping/schema/create` | missing | `enrichment mapping-schema create --file` |
| `/enrichment/mapping/schema/update` | missing | `enrichment mapping-schema update --file` |
| `/enrichment/mapping/schema/delete` | missing | `enrichment mapping-schema delete --id --force` |
| `/enrichment/mapping/data/list` | missing | `enrichment mapping-data list` |
| `/enrichment/mapping/data/upsert` | missing | `enrichment mapping-data upsert --file` |
| `/enrichment/mapping/data/delete` | missing | `enrichment mapping-data delete --file` |
| `/enrichment/mapping/data/download` | missing | `enrichment mapping-data download` |
| `/enrichment/mapping/data/upload` | missing | `enrichment mapping-data upload --file` |
| `/enrichment/mapping/data/truncate` | defer | Destructive bulk operation. |

### Incident management

See the first implementation slice above for lifecycle gaps.

Additional incident gaps:

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/incident/custom-action/do` | defer | `incident custom-action do` |
| `/incident/war-room/create` | implemented | `incident war-room create` |
| `/incident/war-room/delete` | implemented | `incident war-room delete` |
| `/incident/war-room/list` | implemented | `incident war-room list` |
| `/incident/war-room/detail` | implemented | `incident war-room get` |
| `/incident/post-mortem/info` | missing | `postmortem get` |
| `/incident/post-mortem/delete` | missing | `postmortem delete --id --force` |

### Insights

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/insight/account` | missing | `insight account` |
| `/insight/team/export` | missing | `insight team export` |
| `/insight/channel/export` | missing | `insight channel export` |
| `/insight/responder/export` | missing | `insight responder export` |
| `/insight/incident/export` | missing | `insight incidents export` |

### Routes

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/route/list` | missing | `route list` |
| `/route/info` | missing | `route get` |
| `/route/upsert` | missing | `route upsert --file` |

### Schedules

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/schedule/infos` | covered indirectly | Used for enrichment; no command. |
| `/schedule/self` | missing | `oncall schedule self` |
| `/schedule/preview` | missing | `oncall schedule preview --file` |
| `/schedule/create` | missing | `oncall schedule create --file` |
| `/schedule/update` | missing | `oncall schedule update --file` |
| `/schedule/delete` | missing | `oncall schedule delete --id --force` |

### Status page

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/status-page/change/list` | partial | Current `statuspage changes` uses active-list behavior, not this endpoint. |
| `/status-page/change/info` | missing | `statuspage change get` |
| `/status-page/change/update` | missing | `statuspage change update` |
| `/status-page/change/delete` | missing | `statuspage change delete --force` |
| `/status-page/change/timeline/update` | missing | `statuspage timeline update` |
| `/status-page/change/timeline/delete` | missing | `statuspage timeline delete --force` |
| `/status-page/subscriber/list` | missing | `statuspage subscriber list` |
| `/status-page/subscriber/export` | missing | `statuspage subscriber export` |
| `/status-page/subscriber/import` | missing | `statuspage subscriber import --file` |

### Templates

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/template/list` | missing | `template list` |
| `/template/info` | partial | `template get-preset` fetches only the system preset template. |
| `/template/create` | missing | `template create --file` |
| `/template/update` | missing | `template update --file` |
| `/template/delete` | missing | `template delete --id --force` |

### Webhook history

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/webhook/history/list` | missing | `webhook history list` |
| `/webhook/history/detail` | missing | `webhook history get` |

## Platform gaps

### Audit logs

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/audit/operation/list` | missing | `audit operations` |

### Members

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/member/info` | partial | `whoami` calls current member info; no `member info` command. |
| `/person/infos` | covered indirectly | Used for enrichment; no direct lookup command. |
| `/member/invite` | missing | `member invite` |
| `/member/info/reset` | missing | `member update` |
| `/member/delete` | missing | `member delete --id --force` |
| `/member/role/grant` | missing | `member role grant` |
| `/member/role/revoke` | missing | `member role revoke` |
| `/member/role/update` | missing | `member role update` |

### Roles and permissions

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/role/list` | missing | `role list` |
| `/role/info` | missing | `role get` |
| `/role/upsert` | missing | `role upsert --file` |
| `/role/delete` | missing | `role delete --id --force` |
| `/role/enable` | missing | `role enable --id` |
| `/role/disable` | missing | `role disable --id` |
| `/role/permission/list` | missing | `role permissions` |
| `/role/permission/factor/list` | missing | `role permission-factors` |
| `/role/member/grant` | missing | `role member grant` |
| `/role/member/revoke` | missing | `role member revoke` |

## Monitors gaps

The CLI has no first-class `monit` or `monitor` command family yet. Treat the
whole Monitors OpenAPI as missing except for internal SDK support for
`/monit/rule/counter/status`, which currently has no command.

### Datasources

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/monit/datasource/list` | missing | `monit datasource list` |
| `/monit/datasource/info` | missing | `monit datasource get` |
| `/monit/datasource/create` | missing | `monit datasource create --file` |
| `/monit/datasource/update` | missing | `monit datasource update --file` |
| `/monit/datasource/delete` | missing | `monit datasource delete --id --force` |
| `/monit/datasource/sls/projects` | missing | `monit datasource sls-projects` |
| `/monit/datasource/sls/logstores` | missing | `monit datasource sls-logstores` |

### Rules

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/monit/rule/list/basic` | missing | `monit rule list` |
| `/monit/rule/info` | missing | `monit rule get` |
| `/monit/rule/create` | missing | `monit rule create --file` |
| `/monit/rule/update` | missing | `monit rule update --file` |
| `/monit/rule/update/fields` | missing | `monit rule update-fields --file` |
| `/monit/rule/delete` | missing | `monit rule delete --id --force` |
| `/monit/rule/delete/batch` | missing | `monit rule delete-batch --file --force` |
| `/monit/rule/move` | missing | `monit rule move` |
| `/monit/rule/import` | missing | `monit rule import --file` |
| `/monit/rule/export` | missing | `monit rule export` |
| `/monit/rule/status` | missing | `monit rule status` |
| `/monit/rule/dstypes` | missing | `monit rule dstypes` |
| `/monit/rule/audits` | missing | `monit rule audits` |
| `/monit/rule/audit/detail` | missing | `monit rule audit get` |
| `/monit/rule/counter/status` | missing | `monit rule counter status` |
| `/monit/rule/counter/total` | missing | `monit rule counter total` |
| `/monit/rule/counter/channel` | missing | `monit rule counter channel` |
| `/monit/rule/counter/node` | missing | `monit rule counter node` |

### Rulesets

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/monit/store/ruleset/list` | missing | `monit ruleset list` |
| `/monit/store/ruleset/info` | missing | `monit ruleset get` |
| `/monit/store/ruleset/create` | missing | `monit ruleset create --file` |
| `/monit/store/ruleset/update` | missing | `monit ruleset update --file` |
| `/monit/store/ruleset/delete` | missing | `monit ruleset delete --id --force` |

## RUM gaps

The CLI has no first-class `rum` command family yet.

### Applications

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/rum/application/list` | missing | `rum application list` |
| `/rum/application/info` | missing | `rum application get` |
| `/rum/application/infos` | missing | `rum application get-batch` |
| `/rum/application/create` | missing | `rum application create --file` |
| `/rum/application/update` | missing | `rum application update --file` |
| `/rum/application/delete` | missing | `rum application delete --id --force` |

### Issues

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/rum/issue/list` | missing | `rum issue list` |
| `/rum/issue/info` | missing | `rum issue get` |
| `/rum/issue/update` | missing | `rum issue update` |

### Sourcemaps

| Endpoint | Status | Suggested command family |
| --- | --- | --- |
| `/sourcemap/list` | missing | `rum sourcemap list` |

## Current non-catalog commands

These CLI commands use endpoints or static SDK data that are not present in the
current four official OpenAPI specs above. Keep them, but do not use them as
evidence that official OpenAPI coverage is complete.

| Command | Backing behavior |
| --- | --- |
| `change list` | `/change/list` |
| `change trend` | report endpoint for change trend |
| `insight notifications` | report endpoint for notification trend |
| `statuspage list` | `/status-page/list` |
| `statuspage changes` | `/status-page/change/active/list` |
| `template validate` | `/template/preview` |
| `template variables` | SDK static metadata |
| `template functions` | SDK static metadata |
| `field list` | `/field/list` |
| `whoami` | `/account/info` plus `/member/info` |
