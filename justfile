#!/usr/bin/env just --justfile

# Show available commands
default:
    @just --list

# Run the backend API server
run:
    cd backend && go run ./cmd/api

# Build the backend API binary
build:
    cd backend && go build -o bin/api ./cmd/api
    @echo "✅ Built backend/bin/api"

# Clean up Go module dependencies
tidy:
    cd backend && go mod tidy
    @echo "✅ Dependencies cleaned up"

# Run all tests with race detector
test:
    cd backend && go test -v -race -cover ./...

# Run linter
lint:
    cd backend && golangci-lint run --timeout=5m

# Format code
fmt:
    cd backend && gofumpt -w .

# Run all quality checks (test + lint)
check: fmt test lint
    @echo "✅ All checks passed!"

# Generate code from OpenAPI spec
gen:
    cd backend && oapi-codegen -config internal/adapters/api/config.yaml ../schema/api.yaml

# Generate wire dependency injection code
wire:
    cd backend/internal/server && wire

# Generate all code (OpenAPI + Wire)
generate: gen wire
    @echo "✅ All code generated!"

# Clean build artifacts
clean:
    rm -rf backend/bin
    @echo "✅ Build artifacts cleaned"

# Database migrations (placeholder)
db-migrate:
    @echo "Will run migrations once database is set up"
    # cd backend && goose -dir db/migrations postgres "postgresql://..." up

# Install development tools
install-tools:
    go install mvdan.cc/gofumpt@latest
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.1
    go install golang.org/x/vuln/cmd/govulncheck@latest
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
    go install github.com/google/wire/cmd/wire@latest
    @echo "✅ Development tools installed!"