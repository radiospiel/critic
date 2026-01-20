### TUI: test changes with an explicit run

Before completing any significant code changes affecting the TUI, run a manual test by inspecting the rendering. Run this in the fixtures repo, as follows:

	cd tests/integration
	make fixtures
	cd  fixtures/repo
	<<run critic>> 

### Ask for human reviewer approval

If a critic or critic2 MCP server is available, but not on claude code for web:

- Before committing any significant code changes, call the get_review_feedback tool with a summary of what you've done, if a critic or critic2 MCP server is available. Wait for reviewer approval before proceeding. Address any feedback in subsequent iterations.

### Test

- When writing tests, use the assert package. If a function is missing in the package, generate one. For example, this

	if !contains(conversations, conv1.ID) {
    	t.Error("expected conv1 in conversations")
	}
	
should use 

    assert.Contains(t, conversations, conv1.ID, "expected %v in conversations %v", conv1.ID, conversations)
	
