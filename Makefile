BINARY := lxspawn
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
GO := GOOS=linux go

.PHONY: build clean test vet lint install

build:
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/lxspawn

clean:
	rm -f $(BINARY)

test:
	$(GO) test -race ./... -v

vet:
	$(GO) vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed"

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
