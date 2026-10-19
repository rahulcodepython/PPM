# Build ppm from source. Needs Go (version in go.mod).
#   make install   build and install to ~/.local/bin (adds it to PATH)
#   make dev       rebuild and install after every change you make
#   make build     just build ./ppm
# Windows without make: see "Build from source" in README.md.

# A plain version number: the tag on HEAD for a release checkout, otherwise
# the version the next release will get (same rules as the release workflow).
VERSION ?= $(shell (git describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null || sh .github/scripts/next-version.sh 2>/dev/null || echo v0.0.0) | sed 's/^v//')
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG     := github.com/rahulcodepython/PPM/internal/cli
LDFLAGS := -s -w -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

.PHONY: build install dev uninstall test lint snapshot clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o ppm ./cmd/ppm

install: build
	./ppm self install

# Same as install; run it again whenever you change the code.
dev: install

uninstall:
	ppm self uninstall

test:
	go test ./...

lint:
	gofmt -l . | (! grep .) && go vet ./...

# Full release build into ./dist without publishing (needs goreleaser).
snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf ppm ppm.exe dist man completions
