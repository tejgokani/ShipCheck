VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := bin/shipcheck
MODULE  := github.com/tejgokani/shipcheck
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all build test lint vet clean release-dry install-hooks help

all: build

## build: Compile the shipcheck binary
build:
	@mkdir -p bin
	go build $(LDFLAGS) -o $(BINARY) .

## test: Run all tests with race detector and coverage
test:
	go test ./... -race -coverprofile=coverage.out
	@echo ""
	@go tool cover -func=coverage.out | tail -1

## test-short: Run tests without race detector (faster)
test-short:
	go test ./... -coverprofile=coverage.out

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format source code
fmt:
	gofmt -s -w .
	goimports -w . 2>/dev/null || true

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out dist/

## release-dry: Dry-run goreleaser release (snapshot, no publish)
release-dry:
	goreleaser release --snapshot --clean

## install-hooks: Install post-session hooks for Claude Code and Cursor
install-hooks:
	@echo "Installing shipcheck hooks..."
	@chmod +x scripts/hooks/claude-code-hook scripts/hooks/cursor-hook
	@mkdir -p ~/.claude/hooks
	@cp scripts/hooks/claude-code-hook ~/.claude/hooks/post-session
	@chmod +x ~/.claude/hooks/post-session
	@echo "Claude Code hook installed to ~/.claude/hooks/post-session"
	@echo ""
	@echo "For Cursor, add this to your Cursor settings:"
	@echo "  \"terminal.integrated.shellArgs.osx\": [\"-c\", \"source ~/.cursor-hook\"]"
	@cp scripts/hooks/cursor-hook ~/.cursor-hook 2>/dev/null || true
	@echo ""
	@echo "Hooks installed. Run 'shipcheck' after your next session."

## install: Install shipcheck binary to /usr/local/bin
install: build
	@echo "Installing shipcheck to /usr/local/bin..."
	@cp $(BINARY) /usr/local/bin/shipcheck
	@echo "Done. Run 'shipcheck --help' to get started."

## uninstall: Remove shipcheck binary from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/shipcheck
	@echo "shipcheck uninstalled."

## tidy: Tidy go module dependencies
tidy:
	go mod tidy

## help: Show this help message
help:
	@echo "shipcheck Makefile"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'
