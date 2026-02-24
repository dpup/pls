VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint vet fmt fix security clean

## build: compile the pls binary
build:
	go build $(LDFLAGS) -o pls .

## test: run all tests
test:
	go test ./...

## lint: run golangci-lint (install: https://golangci-lint.run/welcome/install/)
lint: vet
	@which golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not installed, skipping"; exit 0; }
	golangci-lint run ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: run gofmt -s (simplify) on all Go files
fmt:
	gofmt -s -w .

## fix: auto-format and tidy
fix: fmt
	go mod tidy

## security: run govulncheck for known vulnerabilities
security:
	@which govulncheck > /dev/null 2>&1 || { echo "install: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	govulncheck ./...

## clean: remove build artifacts
clean:
	rm -f pls

## all: lint, test, build
all: lint test build
