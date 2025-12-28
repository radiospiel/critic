all: test integration

test:
	go test ./...

integration:
	make -C tests/integration/
