# Building `filters` conditions — shared reference

Read this card BEFORE composing any `filters`, `source_filters`, or
`target_filters` value (silence / inhibit / drop rules in
`reference/noise.md`, escalation rules in `reference/escalation.md`). These
fields share one shape and one key vocabulary — and a wrong key is the worst
kind of error: the server may accept the rule, yet it silently never matches
anything.

## Shape — OR of ANDs

The value is an OR-of-AND condition tree: the outer array holds AND groups,
the rule fires when ANY group matches; each inner array holds
`{key, oper, vals}` conditions that must ALL match.

```json
[
  [{"key":"severity","oper":"IN","vals":["Critical"]},
   {"key":"labels.service","oper":"IN","vals":["payments-api"]}],
  [{"key":"labels.env","oper":"IN","vals":["staging"]}]
]
```

→ (Critical AND service=payments-api) OR (env=staging).

## Operators and values

- `oper` is `IN` (the object's value for `key` must equal one of `vals`) or
  `NOTIN` (must equal none of them).
- `vals` entries also accept `/regex/` patterns, e.g.
  `{"key":"title","oper":"IN","vals":["/timeout|connection refused/"]}`.
- **Missing-key trap**: when the object does not carry `key` at all, `IN`
  never matches — and `NOTIN` ALWAYS matches. A `NOTIN` condition on a
  misspelled key doesn't narrow the rule; it silently matches everything.

## Keys — use the canonical names only

Common to every rule family: `severity`, `status`, `title`, `description`,
`data_source_id`, and `labels.<name>` for any custom label. Per family:

| rule family | matched against | extra keys | keys that DO NOT exist here |
|---|---|---|---|
| silence / drop (`filters`), inhibit (`source_filters` / `target_filters`), alert pipeline (`rules[].if`, `alert_inhibit.source_filters`) | each alert event | `alert_key`, `title_rule` | `dedup_key` |
| escalation (`filters`) | the incident | `dedup_key` | `alert_key`, `title_rule` |

A key outside the family's vocabulary (e.g. `dedup_key` in a silence rule)
produces a rule that never matches while looking configured — check the table
before creating, and prefer server-rejected over silently-dead if unsure.

**Legacy aliases — never use them.** `event_severity`, `alert_severity`,
`incident_severity`, `alert_status`, `incident_status`, and `integration_id`
are stored aliases that always carry the SAME value as the canonical
`severity` / `status` / `data_source_id` on their surface. Spelling them adds
no precision and invites wrong reads — on escalation rules `alert_severity`
is an alias of the *incident's* severity, not of any alert's. Always write
the canonical key.

## Building filters from incident labels (scoping a rule to one incident)

To scope a rule to one incident's blast radius, build a single AND group from
that incident's own data (`fduty incident detail <incident-id>`):

1. Start the group with a severity condition:
   `{"key":"severity","oper":"IN","vals":["<incident_severity>"]}`.
2. For each entry in the incident's `labels` object, add one condition
   `{"key":"labels.<label-key>","oper":"IN","vals":["<label-value>"]}` — but
   only when the label is actually distinguishing. Drop a label if its value
   is:
   - purely numeric (any kind of ID — `instance_id`, `pod_id`, …),
   - longer than 256 characters (embedded JSON, stack traces, long text),
   - a date/time value (`2026-07-01T10:00:00Z`, unix timestamps, …), or
   - under a generically noisy key regardless of value — e.g.
     `trigger_value`, `prom_ql`, `detail_url`, any `*_url` key,
     `first_trigger_time`, other `*timestamp*` keys, `rule_config`.
3. After creating the rule, confirm it with the matching `*-rule-list`
   command and read back its `filters` to make sure the conditions
   round-tripped as intended.
