BINARY  := pik
MODULES := opendcim spip_saisies
VERSION ?= dev
LDFLAGS := -s -w -X github.com/Chocapikk/pik/pkg/cli.Version=$(VERSION)

.PHONY: build static standalone vet test

build:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pik/

static:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pik/

standalone: build
	@for m in $(MODULES); do ln -sf $(BINARY) bin/$$m; done

vet:
	go vet ./...

test:
	go test ./...
