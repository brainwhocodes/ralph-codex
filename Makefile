# Lisa Loop - Makefile

.PHONY: build install test test-integration lint clean help all

# Variables
BINARY=lisa
CMD_PATH=./cmd/lisa
GO=go
GOFLAGS=-v

# Default target
all: build test lint

## build: Build the lisa binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) $(CMD_PATH)

## install: Install lisa to $GOPATH/bin and skills/templates
install: install-bin install-templates install-skills

## install-bin: Install Go binary
install-bin:
	$(GO) install $(GOFLAGS) $(CMD_PATH)

## install-templates: Install project templates to ~/.lisa/templates
install-templates:
	@echo "Installing templates to ~/.lisa/templates..."
	@mkdir -p ~/.lisa/templates
	@cp -r templates/* ~/.lisa/templates/
	@echo "Templates installed successfully"

## install-skills: Install Codex skills to ~/.codex/skills
install-skills:
	@echo "Installing Codex skills to ~/.codex/skills..."
	@mkdir -p ~/.codex/skills
	@cp .codex/skills/* ~/.codex/skills/
	@chmod +x ~/.codex/skills/*
	@echo "Codex skills installed successfully"
	@echo "Available skills:"
	@ls -1 ~/.codex/skills/ | grep -v '^\.'

## test: Run all tests
test:
	$(GO) test ./...

## test-integration: Run integration tests
test-integration:
	$(GO) test -tags=integration ./...

## test-verbose: Run tests with verbose output
test-verbose:
	$(GO) test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run golangci-lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.50.1"; \
		exit 1; \
	fi

## fmt: Format code with gofmt
fmt:
	$(GO) fmt ./...

## vet: Run go vet
vet:
	$(GO) vet ./...

## clean: Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -f /tmp/lisa-test-*

## run: Build and run lisa
run: build
	./$(BINARY) --help

## setup-test: Create test project
setup-test:
	./$(BINARY) setup --name test-project

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## deps-update: Update dependencies
deps-update:
	$(GO) get -u ./...
	$(GO) mod tidy

## help: Show this help message
help:
	@echo "Lisa Loop - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all              Build, test, and lint"
	@echo "  build            Build the lisa binary"
	@echo "  install          Install lisa, templates, and skills"
	@echo "  install-bin      Install Go binary only"
	@echo "  install-templates Install project templates only"
	@echo "  install-skills   Install Codex skills to ~/.codex/skills"
	@echo "  test             Run all tests"
	@echo "  test-integration Run integration tests"
	@echo "  test-verbose     Run tests with verbose output"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Format code with gofmt"
	@echo "  vet              Run go vet"
	@echo "  clean            Clean build artifacts"
	@echo "  run              Build and run lisa"
	@echo "  setup-test       Create test project"
	@echo "  deps             Download dependencies"
	@echo "  deps-update      Update dependencies"
	@echo "  help             Show this help message"
