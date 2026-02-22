.PHONY: test test-verbose clean api-tests

# Run integration tests with required -p 1 flag
test: fixtures api-tests
	go test -p 1 -v

# Run API integration tests (bash-based, using testsh)
api-tests:
	$(MAKE) -C api test

# Run integration tests with verbose output
# test-verbose:
# 	go test -p 1 -v

# Run with coverage
test-coverage:
	go test -p 1 -cover

# Clean test cache
clean:
	go clean -testcache
	rm -rf fixtures/repo

fixtures: fixtures/repo/created_at

fixtures/repo/created_at: fixtures/v*.txt fixtures/make_repo.sh
	fixtures/make_repo.sh
	touch fixtures/repo/created_at
