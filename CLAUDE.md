- Before completing any significant code changes, call get_review_feedback with
a summary of what you've done. Wait for reviewer approval before proceeding.
Address any feedback in subsequent iterations.
- When writing tests, use the assert package. If a function is missing in the package, generate one. For example, this

	if !contains(conversations, conv1.ID) {
    	t.Error("expected conv1 in conversations")
	}
	
should use 

    assert.Contains(t, conversations, conv1.ID, "expected %v in conversations %v", conv1.ID, conversations)
	
