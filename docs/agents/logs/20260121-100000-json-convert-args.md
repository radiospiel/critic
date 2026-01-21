# Task: JSON convert args for SetDiffArgs

**Started:** 2026-01-21 10:00:00
**Ended:** 2026-01-21 10:15:00
**Strategy:** Refactoring
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus

## Objective
Refactor SetDiffArgs (and similar functions) to use JSON marshaling/unmarshaling to convert structs to map[string]any, eliminating manual type conversion loops.

Current pattern (verbose):
```go
bases := make([]any, len(args.Bases))
for i, b := range args.Bases {
    bases[i] = b
}
// ... repeat for each field
```

Target pattern (simple):
```go
m := observable.StructToMap(args)
s.SetValueAtKey(KeyDiffArgs, m)
```

## Progress
- [x] Explore current implementation
- [x] Read strategy guide
- [x] Ensure test coverage exists
- [x] Add StructToMap helper to observable package
- [x] Add tests for StructToMap (5 test cases)
- [x] Refactor SetDiffArgs to use StructToMap
- [x] Analyze other functions - not refactored due to missing JSON tags
- [x] Run tests to verify no regressions
- [x] Commit and push

## Obstacles
- **Issue:** Other functions (SetResolvedBases, SetConversationsForFile, SetConversationSummary) use manual key mapping with different casing than the struct field names, and the source structs lack JSON tags.
  **Resolution:** Left these functions as-is. The StructToMap helper requires JSON tags for proper key casing. Adding JSON tags to the structs would be out of scope for this refactoring.

## Outcome
Added `StructToMap[T]` generic function to observable package with comprehensive tests. Refactored `SetDiffArgs` from 20 lines to 8 lines by removing manual slice-to-[]any conversion loops.

## Learnings
- The `StructToMap` function is the inverse of the existing `GetValueAs[T]` function
- JSON tags on structs are required for `StructToMap` to produce the correct key casing
- The round-trip (StructToMap -> SetValueAtKey -> GetValueAs) was verified to work correctly
