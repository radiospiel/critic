### VS Code Extension

The extension source is in `editors/vscode/`. To install after changes:

```
cd editors/vscode && npm run compile && npx vsce package --no-dependencies && code --install-extension critic-vscode-0.1.0.vsix --force
```

After installing, the user must **reload the VS Code window** (Cmd+Shift+P → "Developer: Reload Window") for changes to take effect.

### Task Strategy Selection

Before starting any task, identify which strategy applies from [agents/strategy-guide.md](agents/strategy-guide.md):

- **Bug Fix**: Something is broken, unexpected behavior, errors
- **Feature (TDD)**: New functionality, "add X" requests
- **Refactoring**: Code quality improvements, restructuring
- **Performance**: Optimization, speed/memory issues

**Required workflow:**
1. State which strategy you're following and why
2. Follow that strategy's workflow from the guide
3. If uncertain, ask the human before proceeding
4. For mixed tasks, decompose and apply strategies separately

### Task Progress Logging

Maintain a progress log in `agents/logs/` for each significant task. This provides visibility into agent work and captures insights.

Use the file "agents/logs-template.md" as a template

**Log file naming:** `YYYYMMDD-HHMMSS-short-description.md` (e.g., `20250115-143022-fix-scroll-crash.md`)

**To estimate complexity**, use the following guidance:
- Simple: Task could be completed without any critical human feedback
- Medium: A planning stage was necessary, with important human feedback. Human feedback after the planning stage was mostly cosmetic.
- Complex: The initial plan was not sufficient to guide the agent to a successful outcome, repeated human interventions had been necessary.

Note that timestamps **must always** have the time of day! It is important to always update the "Ended" timestamp when committing work.

**When to log:**
- Create the log when starting a non-trivial task
- Update progress as you complete steps
- Always document obstacles, even if resolved quickly
- when task completes:
	- Finalize with outcome 
	- update the header sectionn.

**Why obstacles matter:** Documenting obstacles helps identify recurring issues, improves future estimates, and provides context if the task is handed off or revisited.

### Ask for human reviewer approval

If the "critic" MCP server is available, but not on claude code for web:

- Before committing any significant code changes, call the get_review_feedback tool. Wait for reviewer approval before proceeding. Address any feedback in subsequent iterations.

### Test

- When writing tests, use the assert package. If a function is missing in the package, generate one. For example, this

	if !contains(conversations, conv1.ID) {
    	t.Error("expected conv1 in conversations")
	}
	
should use 

    assert.Contains(t, conversations, conv1.ID, "expected %v in conversations %v", conv1.ID, conversations)
