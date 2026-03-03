---
name: critic-summarize
description: Post explanations for the changes since master
allowed-tools: Bash
---

Review all changes since master. Post explanations for any complex or surprising implementation detail via

```bash
critic agent explain "path/to/file" lineno "your explanation>"
```
