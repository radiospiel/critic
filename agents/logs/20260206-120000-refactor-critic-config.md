# Task: Refactor .critic configuration files into project.critic YAML

**Started:** 2026-02-06 12:00:00
**Ended:** 2026-02-06 13:00:00
**Strategy:** Feature (TDD) + Refactoring
**Status:** Complete
**Complexity:** Complex
**Used Models:** Opus

## Objective
Replace .criticignore and .critictest with a unified project.critic YAML file. Implement git pathspec glob matching, wire into CLI/server, update frontend, and optimize git diff operations with path filtering.

## Progress
- [x] Explored codebase and understood current architecture
- [x] Created task plan
- [x] Add YAML dependency (gopkg.in/yaml.v3)
- [x] Implement git pathspec glob matching with tests (src/config/pathspec.go)
- [x] Implement project config YAML parser with tests (src/config/project.go)
- [x] Create project.critic config file
- [x] Add GetProjectConfig proto RPC method and regenerate Go + TS
- [x] Wire project config into CLI (--project flag) and server
- [x] Pass paths to git diff/ls-files commands (optimized at source)
- [x] Implement GetProjectConfig API handler
- [x] Update frontend FileList.tsx to use GetProjectConfig API
- [x] Remove .criticignore and .critictest
- [x] Run tests and verify (all pass)

## Obstacles
- protoc not installed: used buf CLI instead for proto generation
- Frontend dist not built: created placeholder for incremental Go builds
- pathspec regex for basename matching: initial `^(?:.*/?)?` prefix was too permissive, fixed to `^(?:.*/)?`

## Outcome
Successfully refactored configuration into project.critic YAML format. Added editor config support, git pathspec glob matching with comprehensive tests, and a proper proto RPC endpoint for frontend config loading.
