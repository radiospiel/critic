.PHONY: test test-verbose clean api-tests

# Run integration tests with required -p 1 flag
test: fixtures api-tests
	go test -p 1 -v

# Run API integration tests (bash-based, using testsh)
# Reuse repo-root binary from CI if available, otherwise build locally
REPO_CRITIC_BIN := $(wildcard ../../critic)
api-tests:
	$(MAKE) -C api test $(if $(REPO_CRITIC_BIN),CRITIC_BIN=$(abspath $(REPO_CRITIC_BIN)))

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
