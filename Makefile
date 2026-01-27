BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
GOBIN := $(shell go env GOPATH)/bin
OS := $(shell uname -s)

# Proto source and generated files
PROTO_DIR := src/api/proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_GEN_GO := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/%.pb.go)
PROTO_GEN_CONNECT := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/apiconnect/%.connect.go)

.PHONY: all build test unit-tests integration install uninstall clean install-deps install-buf install-protoc proto

all: test integration

build: .install-deps.mtime $(PROTO_GEN_GO)
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
	elif [ "$(OS)" = "Darwin" ]; then \
		echo "Installing protoc via Homebrew..."; \
		brew install protobuf; \
	elif [ "$(OS)" = "Linux" ]; then \
		echo "Installing protoc via apt..."; \
		sudo apt-get update && sudo apt-get install -y protobuf-compiler; \
	else \
		echo "Unknown OS: $(OS). Please install protoc manually."; \
		exit 1; \
	fi

# Generate .pb.go and .connect.go from .proto files
src/api/%.pb.go src/api/apiconnect/%.connect.go: $(PROTO_DIR)/%.proto
	protoc -I $(PROTO_DIR) \
		--go_out=src/api --go_opt=paths=source_relative \
		--connect-go_out=src/api/apiconnect --connect-go_opt=paths=source_relative \
		$<

# Convenience target to regenerate all proto files
proto: $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT)
