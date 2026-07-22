#!/usr/bin/env bash
# incident-summary.sh <incident-id> — one-shot, read-only fault-analysis fetch.
#
# A full incident summary needs six different commands (detail does NOT bundle them).
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
