BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin

# Proto source and generated files
PROTO_DIR := src/api/proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_GEN_GO := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/%.pb.go)
PROTO_GEN_CONNECT := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/apiconnect/%.connect.go)

# Frontend
FRONTEND_DIR := src/webui/frontend
FRONTEND_DIST := src/webui/dist

.PHONY: all build test unit-tests integration install uninstall clean install-deps proto proto-ts frontend

all: tests

build: install-deps proto build-server frontend

build-server: $(BINARY)

$(BINARY): proto
	go build -o $(BINARY) ./src/cmd
	
# Build frontend (React app)

frontend: proto $(FRONTEND_DIST)/index.html

$(FRONTEND_DIST)/index.html: $(FRONTEND_DIR)/package.json $(shell find $(FRONTEND_DIR)/src -type f 2>/dev/null)
	cd $(FRONTEND_DIR) && npm install && npm run build

# building the frontend is required because the server embeds the frontend
tests: frontend unit-tests integration-tests

unit-tests:
	LOG_FILE=/tmp/critic.test go test $$(go list ./... | grep -v '/tests/')

integration-tests:
	LOG_FILE=/tmp/critic.test make -C tests/integration/

# Installation
install: build
	install -d $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -f .install-deps.mtime
	rm -rf $(FRONTEND_DIST)

install-deps: .install-deps.mtime

.install-deps.mtime: scripts/install-deps
	./scripts/install-deps
	@touch $@

# Generate .pb.go and .connect.go from .proto files
# Supports both protoc and buf (buf is preferred when available)
src/api/%.pb.go src/api/apiconnect/%.connect.go: $(PROTO_DIR)/%.proto
	echo "Using protoc to generate protobuf code..."; \
	protoc -I $(PROTO_DIR) \
		--go_out=src/api --go_opt=paths=source_relative \
		--connect-go_out=src/api --connect-go_opt=paths=source_relative \
		$<; \

# Convenience target to regenerate all proto files
proto: $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT)

# Generate TypeScript types for frontend using buf
proto-ts:
	cd $(FRONTEND_DIR) && npm install && npx buf generate ../../api/proto
