#!/bin/bash
# open-in-iterm.sh

PROFILE="${ITERM_PROFILE:-Default}"
COMMAND="$1"

osascript <<EOF
tell application "iTerm"
    activate
    create window with profile "$PROFILE"
    tell current session of current window
        write text "$COMMAND"
    end tell
end tell
EOF
