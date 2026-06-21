#!/usr/bin/env bash
# incident-summary.sh <incident-id> — one-shot, read-only fault-analysis fetch.
#
# A full incident summary needs six different commands (detail does NOT bundle them).
# This runs all of them and prints the results in one block, so the summary is written
# from real output with nothing to guess or fabricate. Read-only; safe to run anytime.
#
#   usage: bash incident-summary.sh <incident-id>
#
# To tie post-mortems to this incident specifically, re-run the last section with the
# channel_id from "incident detail":  fduty incident post-mortem-list --channel-ids <id>
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

run() { echo "===== fduty $* ====="; fduty "$@" --output-format toon 2>&1; echo; }

run incident detail        "$ID"              # ① 详情 + AI summary + alert counts + channel_id
run incident alerts        "$ID"              # ② contributing alerts
run incident timeline      "$ID"              # ④ timeline
run incident similar       "$ID" --limit 5    # ⑤ similar past incidents (channel-backed)
run incident post-mortem-list --limit 10      # ⑥ recent post-mortems (add --channel-ids to scope)
run change list --since 24h                   # ③ correlated changes (shared labels + time)
