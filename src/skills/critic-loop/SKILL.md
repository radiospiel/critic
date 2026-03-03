---
name: critic-loop
description: Resolve all critic conversations iteratively until none remain
allowed-tools: Bash, Read, Edit, Write, Grep, Glob
---

Resolve all unresolved critic conversations. Repeat until done:

1. Run `critic agent list --status unresolved` to find open conversations.
2. If none remain, stop.
3. For each conversation, read it via `critic agent show <uuid>`.
4. Address the feedback — make code changes if needed.
5. Reply via `critic agent reply --author ai <uuid> '<message>'`.
6. Go back to step 1.
