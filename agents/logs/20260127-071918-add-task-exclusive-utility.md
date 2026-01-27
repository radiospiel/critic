# Task: Add task.RunExclusively utility

**Started:** 2026-01-27 07:19:18
**Ended:** 2026-01-27 07:25:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus 4.5
**Token usage (Estimated):** ~10k input, ~5k output

## Objective
Create a task utility in `simple-go/tasks` that allows running functions exclusively by ID:
- `task.RunExclusively(id, taskFunc)` - runs a function exclusively
- If a task with the same ID is already running, return an error
- Returns a `Task` object with a channel to pull results from
- The `Task` object has an `Abort()` method to terminate the task

## Progress
- [x] Create progress log
- [x] Write failing tests for RunExclusively
- [x] Implement Task struct and RunExclusively function
- [x] Implement Abort functionality
- [x] Run tests and verify all pass
- [x] Commit changes

## Obstacles
None

## Outcome
Implemented `simple-go/tasks` package with:
- `RunExclusively[T any](id string, taskFunc func() T)` - run a task exclusively by ID
- `RunExclusivelyWithContext[T any](id string, taskFunc func(*Context) T)` - run with abort-aware context
- `Task[T]` struct with `Done()` channel and `Abort()` method
- `Context` struct with `IsAborted()` for cooperative cancellation

## Insights
- Using variadic for optional reduce parameter was over-engineering; simpler interface is better
