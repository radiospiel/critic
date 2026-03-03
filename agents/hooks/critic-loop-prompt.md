Before finishing, check for unresolved reviewer feedback:

1. Run `critic agent conversations --status=actionable` to find conversations that need your attention (unresolved, with the last message from a human reviewer).
2. If there are actionable conversations, read each one via `critic agent conversation <uuid>`, address the feedback (make code changes if needed), and reply via `critic agent reply <uuid> '<message>'`.
3. If there are no actionable conversations, you may finish.
