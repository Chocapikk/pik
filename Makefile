BINARY  := pik
VERSION ?= dev
LDFLAGS := -s -w -X github.com/Chocapikk/pik/pkg/cli.Version=$(VERSION)

.PHONY: build static install vet test clean

build:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pik/

static:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pik/

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/pik/

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin/
