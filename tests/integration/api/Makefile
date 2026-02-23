.PHONY: test clean

CRITIC_BIN ?= $(shell pwd)/critic

test: $(CRITIC_BIN)
	CRITIC_BIN=$(CRITIC_BIN) bash test_api.sh

$(CRITIC_BIN):
	go build -o $(CRITIC_BIN) ../../../src/cmd

clean:
	rm -f critic
