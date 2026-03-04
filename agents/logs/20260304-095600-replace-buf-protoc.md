# Task: Replace buf with direct protoc invocations (#126)

**Started:** 2026-03-04 09:56:00
**Ended:** 2026-03-04 10:05:00
**Strategy:** Refactoring
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus
**Token usage (Estimated):** 50k input, 10k output

## Objective
Replace buf-based proto generation with direct protoc invocations for TypeScript,
and require system-level installation of protoc-gen-es and protoc-gen-connect-es.

## Progress
- [x] Remove buf.yaml and buf.gen.yaml from root and frontend
- [x] Update Makefile proto-ts target to use protoc directly
- [x] Update install-deps to install protoc-gen-es and protoc-gen-connect-es
- [x] Remove @bufbuild/buf from frontend devDependencies
- [x] Update npm generate script to use protoc

## Obstacles
None.

## Outcome
Removed buf dependency in favor of direct protoc invocations with system-installed plugins.

## Insights
The Go proto generation already used protoc directly; only TS generation relied on buf.
