#!/usr/bin/env bash
# incident-summary.sh <incident-id> — one-shot, read-only fault-analysis fetch.
#
# A full incident summary needs seven different commands (detail does NOT bundle them).
# This runs all of them and prints the results in one block, so the summary is written
# from real output with nothing to guess or fabricate. Read-only; safe to run anytime.
#
#   usage: bash incident-summary.sh <incident-id>
#
# Section ⑥ lists recent post-mortems account-wide. To scope them to THIS incident's
# channel, read channel_id from the projected detail section and re-run:
# fduty incident post-mortem-list --channel-ids <id>
#
# Note: errexit (-e) is intentionally NOT set — every section must run even if one
# command fails, so the summary stays as complete as possible. Each command's own
# errors are captured inline via the `2>&1` in run().
set -uo pipefail

ID="${1:-}"
if [ -z "$ID" ]; then
  echo "usage: bash incident-summary.sh <incident-id>" >&2
  exit 2
fi

# Project detail explicitly because its default table includes unbounded narrative
# fields. The other read verbs use their compact default renderers; raw toon would
# dump every empty field plus heavy blobs like a change's labels.steps.
run() { echo "===== fduty $* ====="; fduty "$@" 2>&1; echo; }

run incident detail        "$ID" --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon # ① 详情 + AI summary + alert counts + channel
run incident alerts        "$ID"              # ② contributing alerts
run incident timeline      "$ID"              # ④ timeline
run incident similar       "$ID" --limit 5    # ⑤ similar past incidents (channel-backed)
run incident post-mortem-list --limit 10      # ⑥ recent post-mortems (add --channel-ids to scope)
run change list --since 24h                   # ③ correlated changes (shared labels + time)

# ⑦ concurrent incidents — every incident in this account, any channel, any progress, whose
# start_time falls within ±15 min of this one's. Alert grouping runs per channel, so one root
# cause spanning several services/channels arrives as several incidents; this is the only
# section that looks sideways at them. The reference incident itself appears in the list.
START_RAW=$(fduty incident detail "$ID" --json 2>/dev/null | jq -r '.start_time // empty')
START_TS=""
if [ -n "$START_RAW" ]; then
  # The CLI renders start_time as RFC3339 in its local zone. GNU date (sandbox / Linux runners)
  # parses it as is; BSD date (macOS runners) wants a literal Z spelled out as an offset and no
  # colon inside the offset.
  NORM=${START_RAW/%Z/+00:00}; NORM=${NORM%:*}${NORM##*:}
  START_TS=$(date -d "$START_RAW" +%s 2>/dev/null || date -j -f '%Y-%m-%dT%H:%M:%S%z' "$NORM" +%s 2>/dev/null)
fi
if [ -n "$START_TS" ]; then
  echo "# ⑦ concurrent incidents: every incident in this account (any channel, any progress) started within ±15 min of $START_RAW — includes $ID itself; the trailing note carries the window total"
  run incident list --since "$((START_TS - 900))" --until "$((START_TS + 900))" --limit 50 --fields incident_id,num,title,incident_severity,progress,start_time,channel_id --output-format toon
else
  echo "===== ⑦ concurrent incidents: SKIPPED — could not read start_time of $ID; run: fduty incident list --since <start-15m> --until <start+15m> --limit 50 --output-format toon ====="
  echo
fi
