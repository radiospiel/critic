# Web UI Implementation Prompt

This document contains the original prompt used to implement the web user interface for Critic.

## Original Prompt

Build a simple webserver which replicates the communication style that we have built in the TUI client. Users must be able to browse the diff, and to comment inline.

The web server will be started using `critic webui`. No authorization is necessary. The web server will also be built in golang.

Use htmx for interactivity. Have the web frontend use websockets to receive updates (file changed, conversation updated etc.) It is sufficient that the websocket only informs the UI that there was an update, the client can then fetch everything itself.

The web server also always runs on the same machine, so it has access to git and all the files.

Make critic start the tui only with a "critic tui" command, rename the ui package to tui.

Copy this prompt into docs/prompts. Add a piece of documentation describing the webui architecture.

## Additional Requirements

- Support light and dark mode in the web UI
