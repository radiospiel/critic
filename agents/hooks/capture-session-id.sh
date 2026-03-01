#!/bin/bash
# capture-session-id.sh
#
# PreToolUse hook that captures the Claude Code session_id from the hook
# payload and stores it in the critic DB via the settings CLI command.
# This fires on any critic MCP tool call.

# Read the hook payload from stdin
PAYLOAD=$(cat)

# Log payload for debugging
echo "$(date): cwd=$(pwd) payload=$PAYLOAD" >> /tmp/critic-hook-debug.log

# Extract session_id from the JSON payload
SESSION_ID=$(echo "$PAYLOAD" | jq -r '.session_id // empty')

if [ -z "$SESSION_ID" ]; then
    exit 0
fi

# Store the session_id in the critic database
critic settings set claude_session_id "$SESSION_ID" 2>/dev/null

exit 0
