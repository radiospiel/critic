Before finishing, check for unresolved reviewer feedback:

1. Call `get_critic_conversations` with status "actionable" to find conversations that need your attention (unresolved, with the last message from a human reviewer).
2. If there are actionable conversations, read each one via `get_full_critic_conversation`, address the feedback (make code changes if needed), and reply via `reply_to_critic_conversation`.
3. If there are no actionable conversations, you may finish.
