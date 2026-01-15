# Testing Guide

This document describes how to run tests for Critic.

## Overview

Critic has multiple levels of testing:

| Type | Location | Purpose |
|------|----------|---------|
| Unit Tests | `*_test.go` files | Test individual functions |
| Integration Tests | `tests/integration/` | Test component interactions |
| E2E Tests | `tests/e2e/` | Test Web UI in browser |

## Go Tests

### Running All Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Running Specific Tests

```bash
# Run tests in a specific package
go test ./internal/tui/...
go test ./internal/git/...

# Run a specific test by name
go test -run TestAnimationTicker ./internal/tui/

# Run tests matching a pattern
go test -run "Test.*Highlight" ./...
```

### Test Packages

| Package | Description |
|---------|-------------|
| `internal/tui` | TUI components (animation, viewstate, ANSI) |
| `internal/git` | Git operations |
| `internal/highlight` | Syntax highlighting |
| `pkg/critic` | Core types and interfaces |

## End-to-End Tests (Puppeteer)

The E2E tests verify the Web UI functionality using Puppeteer (headless Chrome).

### Prerequisites

1. Node.js (v18 or later)
2. npm
3. Built `critic` binary

### Setup

```bash
# Build the critic binary first
go build -o critic ./cmd/critic

# Install test dependencies
cd tests/e2e
npm install
```

### Running E2E Tests

```bash
cd tests/e2e
npm test
```

### Test Output

The tests output results in a colored format:

```
Critic WebUI E2E Tests
==================================================

1. Page Load Tests
--------------------------------------------------
  Main page loads successfully... PASSED
  Page title is correct... PASSED
  Header contains "Critic" text... PASSED
  Theme toggle button exists... PASSED

2. File List Tests
--------------------------------------------------
  File list container exists... PASSED
  File list loads files... PASSED
  ...

==================================================
Results: 15 passed, 0 failed
```

### E2E Test Coverage

The tests cover:

**Page Structure**
- Main page loads with 200 OK
- Correct page title
- Header with "Critic" text
- Theme toggle button present

**File List**
- File list container exists
- Files load via htmx
- File items have status and path elements

**Diff Display**
- Clicking file loads diff
- Diff contains line numbers
- Syntax highlighting elements present

**Theme Toggle**
- Default theme is dark
- Toggle switches to light
- Toggle switches back to dark
- Theme persists in localStorage

**Keyboard Navigation**
- `?` shows help overlay
- `?` again hides help overlay
- `j`/`k` navigate file list

**API Endpoints**
- `/api/files` returns file list HTML
- `/api/diff/{path}` returns diff HTML

**WebSocket**
- htmx WebSocket extension loaded

### Debugging E2E Tests

To run tests with visible browser:

```javascript
// In webui.test.js, change:
const config = {
  headless: false,  // Changed from true
  timeout: 30000,
};
```

### Adding New E2E Tests

Add tests to `tests/e2e/webui.test.js`:

```javascript
await runner.test('My new test', async () => {
  // Navigate or interact
  await page.click('.some-element');

  // Wait for result
  await page.waitForSelector('.expected-element');

  // Assert
  const text = await page.$eval('.expected-element', el => el.textContent);
  assertEqual(text, 'expected value');
});
```

## Integration Tests

Integration tests are in `tests/integration/` and test component interactions.

```bash
# Run integration tests
go test ./tests/integration/...
```

## Continuous Integration

For CI pipelines, run:

```bash
# Go tests
go test -v -race ./...

# E2E tests (requires Xvfb for headless)
cd tests/e2e && npm install && npm test
```

### GitHub Actions Example

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Go Tests
        run: go test -v ./...

      - name: Build
        run: go build -o critic ./cmd/critic

      - name: E2E Tests
        run: |
          cd tests/e2e
          npm install
          npm test
```

## Writing Tests

### Go Test Best Practices

Use the `assert` package as specified in CLAUDE.md:

```go
// Good
assert.Contains(t, conversations, conv1.ID, "expected %v in conversations", conv1.ID)

// Avoid
if !contains(conversations, conv1.ID) {
    t.Error("expected conv1 in conversations")
}
```

### E2E Test Best Practices

1. Use descriptive test names
2. Wait for elements before interacting
3. Clean up state between tests
4. Use timeouts to prevent hanging
5. Test both success and error paths
