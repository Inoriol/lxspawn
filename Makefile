BINARY := lxspawn
SUDO_HELPER := lxspawn-sudo-helper
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test vet lint install

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/lxspawn
	go build $(LDFLAGS) -o $(SUDO_HELPER) ./cmd/lxspawn-sudo-helper

clean:
	rm -f $(BINARY) $(SUDO_HELPER)

test:
	go test ./... -v

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed"

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	install -m 755 $(SUDO_HELPER) /usr/local/bin/$(SUDO_HELPER)
