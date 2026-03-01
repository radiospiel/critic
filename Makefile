BINDIR := $(shell go env GOPATH)/bin
RBINARY := $(BINDIR)/critic
DBINARY := $(BINDIR)/dcritic

# Proto source and generated files
PROTO_DIR := src/api/proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_GEN_GO := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/%.pb.go)
PROTO_GEN_CONNECT := $(PROTO_FILES:$(PROTO_DIR)/%.proto=src/api/apiconnect/%.connect.go)

# Frontend
FRONTEND_DIR := src/webui/frontend
FRONTEND_DIST := src/webui/dist

.PHONY: all build dbuild rbuild test unit-tests integration install uninstall clean install-deps proto proto-ts frontend vscode-extension

all: build tests

build: install-deps debug release

debug: frontend vscode-extension $(DBINARY)
release: frontend vscode-extension $(RBINARY)

GO_FILES := $(shell find src simple-go -name '*.go' -not -name '*_test.go')

# VS Code extension
VSCODE_DIR := editors/vscode
VSCODE_VSIX := src/webui/dist/extensions/critic-vscode.vsix
VSCODE_SRC := $(shell find $(VSCODE_DIR)/src -type f 2>/dev/null) \
              $(VSCODE_DIR)/package.json \
              $(VSCODE_DIR)/tsconfig.json

# Build the VS Code extension .vsix after the frontend (Vite clears dist/ with emptyOutDir)
$(VSCODE_VSIX): $(FRONTEND_DIST)/index.html $(VSCODE_SRC)
	mkdir -p $(dir $(VSCODE_VSIX))
	cp LICENSE $(VSCODE_DIR)/LICENSE
	cd $(VSCODE_DIR) && npm install && npm run compile && \
		node_modules/.bin/vsce package --no-dependencies --out ../../$(VSCODE_VSIX)

reinstall-vscode-extension: vscode-extension
	code --install-extension $(VSCODE_VSIX)

$(RBINARY): $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT) $(GO_FILES) $(VSCODE_VSIX)
	go build -o $(RBINARY) ./src/cmd

$(DBINARY): $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT) $(GO_FILES) $(VSCODE_VSIX)
	go build -gcflags='all=-N -l' -o $(DBINARY) ./src/cmd

# Build frontend (React app)
FRONTEND_SRC := $(shell find $(FRONTEND_DIR)/src -type f 2>/dev/null)

$(FRONTEND_DIST)/index.html: $(FRONTEND_DIR)/package.json $(PROTO_GEN_TS) $(PROTO_GEN_TS_CONNECT) $(FRONTEND_SRC)
	cd $(FRONTEND_DIR) && npm install && npm run build

frontend: $(FRONTEND_DIST)/index.html

# building the frontend is required because the server embeds the frontend
tests: frontend unit-tests integration-tests

unit-tests:
	LOG_FILE=/tmp/critic.test go test $$(go list ./... | grep -v '/tests/')

integration-tests:
	LOG_FILE=/tmp/critic.test make -C tests/integration/

# Installation (binaries are already built into BINDIR, so install is just build)
install: build

uninstall:
	rm -f $(RBINARY) $(DBINARY)

clean: uninstall
	rm -f .install-deps.mtime
	rm -rf $(FRONTEND_DIST)

install-deps: .install-deps.mtime

.install-deps.mtime: scripts/install-deps
	./scripts/install-deps
	@touch $@

# Proto: generated TypeScript files
PROTO_GEN_TS := $(PROTO_FILES:$(PROTO_DIR)/%.proto=$(FRONTEND_DIR)/src/gen/%_pb.ts)
PROTO_GEN_TS_CONNECT := $(PROTO_FILES:$(PROTO_DIR)/%.proto=$(FRONTEND_DIR)/src/gen/%_connect.ts)

# Generate .pb.go and .connect.go from .proto files
src/api/%.pb.go src/api/apiconnect/%.connect.go: $(PROTO_DIR)/%.proto
	protoc -I $(PROTO_DIR) \
		--go_out=src/api --go_opt=paths=source_relative \
		--connect-go_out=src/api --connect-go_opt=paths=source_relative \
		$<

# Generate TypeScript types from .proto files
$(FRONTEND_DIR)/src/gen/%_pb.ts $(FRONTEND_DIR)/src/gen/%_connect.ts: $(PROTO_DIR)/%.proto
	cd $(FRONTEND_DIR) && npx buf generate ../../api/proto

# Regenerate all proto files (Go + TypeScript)
proto: $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT) $(PROTO_GEN_TS) $(PROTO_GEN_TS_CONNECT)

# Convenience aliases
proto-go: $(PROTO_GEN_GO) $(PROTO_GEN_CONNECT)
proto-ts: $(PROTO_GEN_TS) $(PROTO_GEN_TS_CONNECT)
