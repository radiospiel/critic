BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
PROTOC_VERSION := 29.4
GOBIN := $(shell go env GOPATH)/bin

.PHONY: all build test unit-tests integration install uninstall clean install-deps install-buf install-protoc proto

all: test integration

build: .install-deps.mtime
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
	rm -f .install-deps.mtime

.install-deps.mtime: Makefile
	$(MAKE) install-deps
	touch $@

install-deps: install-protoc install-buf
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

install-buf:
	go install github.com/bufbuild/buf/cmd/buf@latest

install-protoc:
	@if command -v protoc >/dev/null 2>&1; then \
		echo "protoc already installed: $$(protoc --version)"; \
	else \
		echo "Installing protoc $(PROTOC_VERSION)..."; \
		curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m).zip"; \
		unzip -o "protoc-$(PROTOC_VERSION)-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m).zip" -d $(GOBIN)/.. bin/protoc; \
		rm "protoc-$(PROTOC_VERSION)-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m).zip"; \
		echo "protoc installed to $(GOBIN)/protoc"; \
	fi

proto:
	protoc -I src/api \
		--go_out=src/api --go_opt=paths=source_relative \
		--connect-go_out=src/api --connect-go_opt=paths=source_relative \
		src/api/critic.proto
