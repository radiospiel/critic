#!/bin/bash
# critic-loop-on-idle.sh
#
# Stop hook guard: prevents infinite re-entry when the critic loop fires.
#
# How it works:
# - First time the Stop hook fires: no guard file exists → create it → exit 0
#   (prompt fires, Claude checks for unresolved critic conversations via MCP)
# - Second time the Stop hook fires (after Claude addressed feedback):
#   guard file exists → remove it → exit 1 (Claude stops normally)
# - Next stop cycle: guard is gone → repeats from the top
#
# This alternating pattern ensures:
# 1. Claude always checks for feedback before going idle
# 2. Claude doesn't loop forever on conversations that remain "unresolved"
#    (since only the human can mark them resolved in the Critic UI)

GUARD_FILE="/tmp/critic-stop-hook-guard"

if [ -f "$GUARD_FILE" ]; then
    # Guard exists from a previous firing — let Claude stop
    rm -f "$GUARD_FILE"
    exit 1
fi

# No guard — this is a fresh stop cycle. Create guard and fire the prompt.
touch "$GUARD_FILE"
exit 0
