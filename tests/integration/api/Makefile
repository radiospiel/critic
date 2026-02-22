.PHONY: test build clean

CRITIC_BIN := critic

test: build
	bash test_api.sh

build: $(CRITIC_BIN)

$(CRITIC_BIN): $(shell find ../../../src ../../../simple-go -name '*.go' -not -name '*_test.go' 2>/dev/null)
	go build -o $(CRITIC_BIN) ../../../src/cmd

clean:
	rm -f $(CRITIC_BIN)
