---
name: critic-step
description: Address unresolved critic feedback from the reviewer
allowed-tools: Bash, Read, Edit, Write, Grep, Glob
---

Address unresolved critic conversations:

1. Run `critic agent list --status unresolved` to find conversations needing attention.
2. For each conversation, read it via `critic agent show <uuid>`.
3. Address the feedback — make code changes if needed.
4. Reply via `critic agent reply --author ai <uuid> '<message>'`.
5. After each reply, re-check with `critic agent list --status unresolved` for new messages and address any new feedback before finishing.
