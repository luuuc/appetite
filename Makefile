.PHONY: build test test-coverage clean install lint ci run

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -ldflags="-s -w -X 'github.com/luuuc/appetite/internal/version.Version=$(VERSION)'"

COVER_PKGS := ./internal/workflow,./internal/cli,./internal/mcp,./cmd/appetite

build:
	go build $(LDFLAGS) -trimpath -o bin/appetite ./cmd/appetite

# test runs the suite with coverage and then gates on the 90% floor
# for the workflow/cli/cmd packages. The pitch makes this gate
# non-negotiable: sub-90% files do not merge. The doc-drift gate
# runs next; it is a graceful no-op when `.doc/definition/` is
# absent (fresh clones, CI runners), and only fires locally where
# the operator keeps the private workspace.
test:
	go test -coverprofile=coverage.out -coverpkg=$(COVER_PKGS) ./...
	go run ./tools/check-coverage coverage.out
	go run ./tools/check-doc-drift

# test-coverage prints the per-function report (useful when the gate
# fails and you want to see exactly which functions are short).
test-coverage:
	go test -coverprofile=coverage.out -coverpkg=$(COVER_PKGS) ./...
	go tool cover -func=coverage.out

clean:
	rm -rf bin/ dist/

install: build
	cp bin/appetite /usr/local/bin/appetite

lint:
	@command -v golangci-lint >/dev/null 2>&1 || \
		(echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)
	@PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run

ci: build test lint
	@echo "All CI checks passed!"

run:
	go run ./cmd/appetite $(ARGS)
