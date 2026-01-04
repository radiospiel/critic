BINARY := critic
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin

.PHONY: all build test integration install uninstall clean

all: test integration

build:
	go build -o $(BINARY) ./cmd/critic

test:
	go test ./...

integration:
	make -C tests/integration/

install: build
	install -d $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
