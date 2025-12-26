# Architectural Decisions

This document records architectural decisions made during the development of Critic.

## Testing Architecture

### 1. Use `must` package for panic-based utilities

**Decision**: Created a separate `internal/must` package for operations that should panic on error rather than return errors.

**Rationale**:
- In integration tests, setup failures should fail fast
- Reduces boilerplate error handling in test setup code
- Clear separation between test assertions (testutils) and utility operations (must)

**Implementation**:
```go
import "git.15b.it/eno/critic/internal/must"

must.WriteFile("test.go", "package main\n")
must.Exec("git", "add", "test.go")
must.Exec("git", "commit", "-m", "add test")
```

**Functions**:
- `Must(err)` - panic if err is not nil
- `Must2(val, err)` - panic if err is not nil, otherwise return val
- `WriteFile(filename, content)` - write file, panic on error (accepts string or []byte)
- `Exec(cmd, args...)` - execute command, panic on error
- `Run(cmd, args...)` - execute command and return output, panic on error
- `MkdirAll(path, perm)` - create directories, panic on error
- `Remove(path)` - remove file, panic on error

### 2. Separate test assertions from panic utilities

**Decision**: Split utilities into two packages:
- `internal/testutils` - test assertions only (AssertEquals, AssertNoError, etc.)
- `internal/must` - panic-based utilities

**Rationale**:
- Single responsibility principle
- testutils is for verifying test outcomes
- must is for setup operations that should never fail
- Clearer API surface

**Usage**:
```go
import (
    "git.15b.it/eno/critic/internal/must"
    tu "git.15b.it/eno/critic/internal/testutils"
)

// Setup with must
must.WriteFile("test.go", content)

// Assertions with tu
tu.AssertNoError(t, err)
tu.AssertEquals(t, actual, expected)
```

### 3. Use `tu` alias for testutils

**Decision**: Import testutils with alias `tu` throughout the codebase.

**Rationale**:
- Shorter, more concise code
- Common pattern in Go testing
- Clear distinction from other test packages

**Implementation**:
```go
import tu "git.15b.it/eno/critic/internal/testutils"

tu.AssertEquals(t, actual, expected)
tu.AssertNoError(t, err)
```

### 4. Integration tests require sequential execution

**Decision**: Integration tests must run with `-p 1` flag (sequential execution).

**Rationale**:
- Tests use `os.Chdir()` to change into temporary git repositories
- Parallel execution would cause directory conflicts
- Each test creates its own temp directory and changes into it

**Enforcement**:
- `TestMain()` prints warning message
- `Makefile` in `tests/integration/` enforces `-p 1` flag
- Documentation in `shared.go` explains requirement

**Usage**:
```bash
cd tests/integration
make test           # Uses -p 1 automatically
go test -p 1 -v     # Manual execution
```

### 5. Shared test setup functions

**Decision**: Centralize git repository setup in `tests/integration/shared.go`.

**Functions**:
- `SetupGitRepo(t)` - creates temp git repo, changes into it, sets up cleanup
- `CommitFile(t, filename)` - commits an existing file

**Rationale**:
- Eliminates code duplication across test files
- Consistent git configuration (user.name, user.email)
- Automatic cleanup via t.Cleanup()

**Usage**:
```go
func TestSomething(t *testing.T) {
    SetupGitRepo(t)  // Creates temp repo, changes into it

    must.WriteFile("test.go", "package main\n")
    CommitFile(t, "test.go")

    // Test runs in temp directory
    // Cleanup happens automatically
}
```

### 6. Integration tests in `tests/integration/` directory

**Decision**: Place integration tests in `tests/integration/` (not `tests/integration/git/`).

**Package name**: `package critic_integration`

**Rationale**:
- Simpler directory structure
- Tests can cover multiple packages without deep nesting
- Package name avoids conflicts with internal packages

### 7. WriteFile accepts string or []byte

**Decision**: `must.WriteFile()` accepts `any` type and uses type switching to handle both string and []byte.

**Rationale**:
- Most tests use string content
- Binary file tests need []byte
- Single function is more ergonomic than two separate functions

**Implementation**:
```go
func WriteFile(filename string, content any) {
    var data []byte
    switch v := content.(type) {
    case string:
        data = []byte(v)
    case []byte:
        data = v
    default:
        panic(fmt.Sprintf("unsupported type %T", content))
    }
    // ... write data
}
```

**Usage**:
```go
must.WriteFile("text.txt", "hello world")              // string
must.WriteFile("binary.dat", []byte{0x00, 0x01, 0x02}) // []byte
```

## Git Package Architecture

### 8. Removed executor interface abstraction

**Decision**: Deleted `CommandExecutor` interface and all `*WithExecutor` functions. Use direct `exec.Command` calls instead.

**Rationale**:
- Integration tests test real git operations
- Unit tests moved to integration tests
- Executor abstraction added complexity without value
- Direct exec.Command is simpler and more maintainable

**Before**:
```go
type CommandExecutor interface {
    Run(name string, args ...string) ([]byte, error)
}

func GetDiff(paths []string, mode DiffMode) (*ctypes.Diff, error) {
    return GetDiffWithExecutor(paths, mode, defaultExecutor)
}

func GetDiffWithExecutor(paths []string, mode DiffMode, executor CommandExecutor) (*ctypes.Diff, error) {
    output, err := executor.Run("git", args...)
    // ...
}
```

**After**:
```go
func GetDiff(paths []string, mode DiffMode) (*ctypes.Diff, error) {
    cmd := exec.Command("git", args...)
    output, err := cmd.Output()
    // ...
}
```

### 9. Panic on invalid git output instead of returning errors

**Decision**: When git returns invalid commit hashes, panic instead of returning error.

**Rationale**:
- Git should ALWAYS return valid commit hashes
- Invalid hash indicates catastrophic system failure, not normal error
- Panicking makes debugging easier (shows exact location)
- Integration tests will catch this immediately

**Implementation**:
```go
// Before:
if !validCommitHash.MatchString(base) {
    return "", fmt.Errorf("invalid merge base format: %s", base)
}

// After:
if !validCommitHash.MatchString(base) {
    panic(fmt.Sprintf("git returned invalid merge base format: %s", base))
}
```

## Test Organization

### 10. Split CommitFile into separate WriteFile and CommitFile

**Decision**: Separate file creation from git operations.

**Before**:
```go
CommitFile(t, "test.go", "package main\n")
```

**After**:
```go
must.WriteFile("test.go", "package main\n")
CommitFile(t, "test.go")
```

**Rationale**:
- Single responsibility: WriteFile writes, CommitFile commits
- More flexible (can write multiple files before committing)
- Clearer test intent
- Follows Unix philosophy (do one thing well)

### 11. Makefile for integration tests

**Decision**: Added `tests/integration/Makefile` to standardize test execution.

**Targets**:
- `make test` - Run tests with -p 1 flag
- `make test-coverage` - Run with coverage report
- `make clean` - Clean test cache

**Rationale**:
- Enforces correct test execution flags
- Provides consistent interface
- Easier for contributors to run tests correctly
- Documents test requirements

## Summary

The key architectural principles are:

1. **Fail fast in tests** - Use panic-based utilities (must package) for operations that should never fail
2. **Separate concerns** - testutils for assertions, must for utilities
3. **Test real implementations** - Integration tests use real git, no mocking
4. **Simple over complex** - Removed executor abstraction in favor of direct exec.Command
5. **Sequential execution** - Integration tests cannot run in parallel due to os.Chdir()
6. **Shared setup** - Centralize common test setup in shared.go
7. **Ergonomic APIs** - Use type switching, aliases, and sensible defaults to reduce boilerplate
