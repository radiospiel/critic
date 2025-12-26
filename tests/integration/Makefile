.PHONY: test test-verbose clean

# Run integration tests with required -p 1 flag
test:
	go test -p 1 -v

# Run integration tests with verbose output
# test-verbose:
# 	go test -p 1 -v

# Run with coverage
test-coverage:
	go test -p 1 -cover

# Clean test cache
clean:
	go clean -testcache
