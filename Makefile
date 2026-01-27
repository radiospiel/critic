BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin

# Proto source and generated files
PROTO_DIR := src/api/proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_GEN_GO := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/%.pb.go)
PROTO_GEN_CONNECT := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/apiconnect/%.connect.go)

.PHONY: all build test unit-tests integration install uninstall clean install-deps proto

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

.install-deps.mtime: scripts/install-deps Makefile
	./scripts/install-deps
	touch $@

install-deps:
	./scripts/install-deps

# Generate .pb.go and .connect.go from .proto files
src/api/%.pb.go src/api/apiconnect/%.connect.go: $(PROTO_DIR)/%.proto
	protoc -I $(PROTO_DIR) \
		--go_out=src/api --go_opt=paths=source_relative \
		--connect-go_out=src/api/apiconnect --connect-go_opt=paths=source_relative \
		$<

# Convenience target to regenerate all proto files
proto: $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT)
