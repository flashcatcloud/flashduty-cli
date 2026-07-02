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
# channel, read its channel_id (fduty incident info --incident-id <id> --output-format
# toon | grep '^channel_id:') and re-run: fduty incident post-mortem-list --channel-ids <id>
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

# Print each command's DEFAULT renderer (a curated table/summary that projects the
# summary-relevant fields), NOT --output-format toon: toon dumps the full raw objects
# — every empty field plus heavy blobs like a change's labels.steps — which overflowed
# the output cap and forced repeated paging. For these read verbs the lean default IS
# the field projection a fault summary needs (id/severity/status/title/channel/times/…).
run() { echo "===== fduty $* ====="; fduty "$@" 2>&1; echo; }

run incident detail        "$ID"              # ① 详情 + AI summary + alert counts + channel
run incident alerts        "$ID"              # ② contributing alerts
run incident timeline      "$ID"              # ④ timeline
run incident similar       "$ID" --limit 5    # ⑤ similar past incidents (channel-backed)
run incident post-mortem-list --limit 10      # ⑥ recent post-mortems (add --channel-ids to scope)
run change list --since 24h                   # ③ correlated changes (shared labels + time)
