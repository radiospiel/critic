# Fix: Startup and UI improvements

**Started:** 2026-02-02 04:15:32
**Ended:** 2026-02-02 04:22:00
**Status:** Complete
**Complexity:** Simple

## Problems

1. Running `./critic` (without arguments) shows log messages about git/mergebase operations before displaying the help screen
2. Help screen lists "Available commands" twice
3. Webui defaults to "Conversations" filter even when there are none

## Root Causes

1. In `src/cli/httpd.go:89`, `getDefaultBases()` was called during command construction, executing git commands even when just showing help
2. The root command's Long description manually listed commands, duplicating Cobra's auto-generated list
3. FileList component always defaulted to 'conversations' filter regardless of whether any exist

## Solutions

1. Defer `getDefaultBases()` call to command execution time
2. Remove manual command list from root command Long description
3. After loading conversation summaries, switch to 'files' filter if no conversations exist

## Changes

- `src/cli/httpd.go`: Lazy initialization of diffBases in RunE
- `src/cli/parser.go`: Removed duplicate "Available commands" from Long description
- `src/webui/frontend/src/components/FileList.tsx`: Default to 'files' filter when no conversations
