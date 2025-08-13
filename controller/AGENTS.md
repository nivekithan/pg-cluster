# PostgreSQL Cluster Operator Agent Guide

## Build & Test Commands
- **Build**: `make build` (builds manager binary)
- **Test**: `make test` (full test suite with linting)
- **Test Single**: `go test ./internal/controller/... -run TestSpecificTest`
- **Test E2E**: `make test-e2e` (requires kind cluster)
- **Lint**: `make lint` (runs golangci-lint)
- **Lint Fix**: `make lint-fix` (runs golangci-lint with auto-fixes)

## Code Style & Standards
- **Language**: Go 1.24.0, Kubernetes Operator pattern
- **Formatting**: Uses `gofmt` and `goimports` formatters
- **Imports**: Standard Go grouping - stdlib, external packages, internal packages
- **Types**: Use typed clients, avoid `interface{}` without good reason
- **Naming**: CamelCase for exported, lowercase for internal, descriptive names
- **Controllers**: Follow controller-runtime patterns, use `ctrl.Result` properly
- **Errors**: Use `fmt.Errorf` for wrapping, log errors before returning
- **Context**: Always pass context down the call chain
- **Testing**: Use Ginkgo/Gomega framework, table-driven tests preferred