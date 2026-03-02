BINARY  := pik
MODULES := opendcim spip_saisies
LDFLAGS := -s -w

.PHONY: build static standalone vet test

build:
	@mkdir -p bin
	go build -o bin/$(BINARY) ./cmd/pik/

static:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pik/

standalone: build
	@for m in $(MODULES); do ln -sf $(BINARY) bin/$$m; done

vet:
	go vet ./...

test:
	go test ./...
