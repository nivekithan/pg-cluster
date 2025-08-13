# PostgreSQL Cluster Operator Agent Guide

## Important Note

The `mariadb-operator/` directory contains a production-ready Kubernetes operator for MariaDB database and serves as a reference for implementing best practices in our PostgreSQL operator. Since both operators manage stateful database applications, the MariaDB operator provides excellent patterns for:

- Custom Resource Definitions (CRDs) and API design
- Controller implementation patterns
- StatefulSet and Pod management
- Backup and recovery mechanisms
- High availability and clustering
- Configuration management
- Webhook implementations
- Testing strategies

**NEVER modify anything in mariadb-operator/** - it's read-only reference material. All PostgreSQL operator development work should be done in the `controller/` directory.

The `postgres-operator/` directory contains the official Crunchy Data PostgreSQL Operator, which is a production-ready reference implementation for PostgreSQL clustering, backup/recovery, and high availability. This serves as an excellent reference for PostgreSQL-specific patterns including:

- PostgreSQL clustering with Patroni
- pgBackRest backup and restore implementation
- WAL archiving and point-in-time recovery
- PostgreSQL configuration management
- Security and RBAC patterns
- Monitoring and observability
- PostgreSQL-specific testing approaches

**NEVER modify anything in postgres-operator/** - it's read-only reference material for learning PostgreSQL operator best practices.

## Build & Test Commands

- **Build**: `cd controller && make build` (builds manager binary)
- **Test**: `cd controller && make test` (full test suite with linting)
- **Test Single**: `cd controller && go test ./internal/controller/... -run TestSpecificTest`
- **Test E2E**: `cd controller && make test-e2e` (requires kind cluster)
- **Lint**: `cd controller && make lint` (runs golangci-lint)
- **Lint Fix**: `cd controller && make lint-fix` (runs golangci-lint with auto-fixes)

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
