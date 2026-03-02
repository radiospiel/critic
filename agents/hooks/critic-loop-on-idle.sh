#!/bin/bash
# critic-loop-on-idle.sh
#
# Stop hook guard: prevents infinite re-entry when the critic loop fires.
#
# How it works:
# - First time the Stop hook fires: no guard file exists → create it → exit 2
#   (Claude continues and checks for unresolved critic conversations via CLI)
# - Second time the Stop hook fires (after Claude addressed feedback):
#   guard file exists → remove it → exit 0 (Claude stops normally)
# - Next stop cycle: guard is gone → repeats from the top
#
# This alternating pattern ensures:
# 1. Claude always checks for feedback before going idle
# 2. Claude doesn't loop forever on conversations that remain "unresolved"
#    (since only the human can mark them resolved in the Critic UI)
#
# Exit codes per https://code.claude.com/docs/en/hooks:
#   0 = allow stop (success, stdout parsed for JSON)
#   2 = block stop (Claude continues, stderr fed back as error message)

GUARD_FILE="/tmp/critic-stop-hook-guard"

if [ -f "$GUARD_FILE" ]; then
    # Guard exists from a previous firing — allow Claude to stop
    rm -f "$GUARD_FILE"
    exit 0
fi

# No guard — this is a fresh stop cycle. Create guard and block the stop,
# sending the prompt via stderr so Claude continues and checks for feedback.
touch "$GUARD_FILE"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cat "$SCRIPT_DIR/critic-loop-prompt.md" >&2
exit 2
