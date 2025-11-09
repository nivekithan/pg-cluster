# pg-cluster

A Kubernetes operator for managing PostgreSQL database clusters with built-in backup, Point-in-Time Recovery (PITR), and easy database provisioning - designed for dynamic environments like PR preview deployments.

## Status

**Work in Progress** - This project is under active development and not yet ready for production use.

## Goals

pg-cluster aims to provide:

- **Database Cluster Management**: Deploy and manage PostgreSQL clusters on Kubernetes
- **Automated Backups**: Integrated pgBackRest for reliable backup management
- **Point-in-Time Recovery (PITR)**: Restore databases to any point in time
- **Easy Database Provisioning**: Quickly create isolated databases within the same cluster
- **PR Preview Optimized**: Designed for dynamic environments where each PR preview needs its own database

## Use Case

The primary use case is enabling ephemeral database instances for PR preview environments. Instead of sharing a single database or manually provisioning databases for each preview, pg-cluster allows you to:

1. Deploy a PostgreSQL cluster once
2. Create isolated databases on-demand for each PR preview
3. Automatically backup and enable PITR for all databases
4. Clean up databases when PR previews are closed

## Architecture

The project consists of two main components:

### 1. Kubernetes Controller (`controllers/pg/`)

A TypeScript-based Kubernetes operator that:
- Watches for `Postgres` custom resources
- Reconciles desired state with actual cluster state
- Manages PostgreSQL deployments, services, and storage
- Built with modern TypeScript patterns using Effect-TS and Zod

### 2. PostgreSQL Image (`images/postgres-with-pgbackrest/`)

A custom PostgreSQL 17 Docker image that includes:
- PostgreSQL 17
- pgBackRest for backup and restore operations
- Custom initialization scripts
- Pre-configured for backup integration

## Project Structure

```
pg-cluster/
├── controllers/pg/          # Kubernetes operator
│   ├── src/
│   │   ├── index.ts        # Main controller loop
│   │   ├── crd.ts          # CRD definition and reconciliation
│   │   └── lib/            # Helper utilities
│   ├── scripts/
│   │   └── generateCrd.ts  # Generate CRD manifests
│   └── generated_manifests/ # Generated Kubernetes manifests
├── images/                  # Docker images
│   └── postgres-with-pgbackrest/
└── patches/                 # pnpm patches
```

## Technology Stack

- **Kubernetes**: Target platform
- **TypeScript**: Controller implementation language
- **Effect-TS**: Functional effect system for robust async workflows
- **Zod**: Runtime type validation and schema definition
- **PostgreSQL 17**: Database engine
- **pgBackRest**: Backup and recovery solution
- **pnpm**: Package manager with workspace support

## Custom Resource Definition

The operator defines a `Postgres` custom resource:

```yaml
apiVersion: kube.nivekithan.com/v1alpha1
kind: Postgres
metadata:
  name: my-postgres-instance
  namespace: test-pg
spec:
  storage: "10Gi"
```

## Development

### Prerequisites

- Node.js (with pnpm)
- Kubernetes cluster (local or remote)
- kubectl configured
- Docker (for building images)

### Setup

1. Install dependencies:
```bash
pnpm install
```

2. Generate CRD manifests:
```bash
cd controllers/pg
pnpm run generate
```

3. Install CRDs to your cluster:
```bash
pnpm run install
```

4. Run the controller:
```bash
pnpm run dev
```

### Building the PostgreSQL Image

```bash
cd images/postgres-with-pgbackrest
./deploy.sh
```

## Current Implementation Status

- [x] Basic CRD definition with storage spec
- [x] Controller watches and reconciles Postgres resources
- [x] Automatic deployment creation for PostgreSQL instances
- [x] Custom PostgreSQL image with pgBackRest
- [ ] Backup configuration and automation
- [ ] Point-in-Time Recovery implementation
- [ ] Database provisioning within existing cluster
- [ ] Service and ingress setup
- [ ] High availability configuration
- [ ] Monitoring and observability
