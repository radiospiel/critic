BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin

.PHONY: all build test unit-tests integration install uninstall clean install-deps

all: test integration

build:
	go build -o $(BINARY) ./src/cmd

test:
	go test ./...

unit-tests:
	go test $$(go list ./... | grep -v '/tests/')

integration:
	make -C tests/integration/

install: build
	install -d $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)

install-deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
