Before finishing, check for unresolved reviewer feedback:

1. Call `get_critic_conversations` with status "unresolved" to list pending feedback.
2. If there are unresolved conversations, call `get_full_critic_conversation` for each one.
3. For each conversation where the last message is from a human reviewer (not from AI), address the feedback: make code changes if needed, then reply via `reply_to_critic_conversation`.
4. If all unresolved conversations already have your reply as the last message, there is nothing new to address — you may finish.
