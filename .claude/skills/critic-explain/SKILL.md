---
name: critic-summarize
description: Summarize all uncommitted changes and post an announcement in the Critic UI
allowed-tools: Bash
---

Review all changes since master. Post explanations for any complex or surprising implementation detail via

```bash
critic agent explain "path/to/file" lineno "your explanation>"
```
