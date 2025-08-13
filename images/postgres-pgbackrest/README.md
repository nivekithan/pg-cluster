# PostgreSQL with pgBackRest

This Docker image combines PostgreSQL 17 with pgBackRest backup tool with intelligent stanza management.

## Features

- **PostgreSQL 17**: Latest PostgreSQL database
- **pgBackRest**: Integrated backup and restore tool
- **Transparent Stanza Management**: Automatically handles database system ID mismatches without user intervention
- **No Persistent Volume Required**: Works seamlessly with ephemeral containers
- **S3-Compatible Storage**: Pre-configured for cloud storage
- **Environment Variable Configuration**: Easy setup with `.env` files
- **Intelligent Wrapper**: pgBackRest commands automatically fix stanza issues behind the scenes

## Quick Start

### 1. Configuration

Create a `.env` file with your S3 storage configuration:

```bash
S3_BUCKET_NAME=your-bucket-name
S3_ENDPOINT=https://your-s3-endpoint.com
S3_ACCESS_KEY=your-access-key
S3_ACCESS_KEY_SECRET=your-secret-key
S3_REGION=your-region
POSTGRES_DB=your-database-name
POSTGRES_USER=your-username
POSTGRES_PASSWORD=your-password
```

### 2. Run

```bash
# Quick start with the run script
./run.sh

# Or manually
docker build -t postgres-pgbackrest .
docker run -d --env-file .env postgres-pgbackrest
```

## pgBackRest Operations

All commands work seamlessly - stanza issues are handled automatically:

```bash
# Check configuration (always works)
docker exec <container> pgbackrest --stanza=my-pg-database check

# Create backup (always works)
docker exec <container> pgbackrest --stanza=my-pg-database backup

# List backups (always works)
docker exec <container> pgbackrest --stanza=my-pg-database info

# Restore database (always works)  
docker exec <container> pgbackrest --stanza=my-pg-database restore
```

## How It Works

The container includes an intelligent pgBackRest wrapper that:

1. **Detects stanza mismatches** automatically
2. **Recreates stanza** when database system ID changes  
3. **Retries commands** seamlessly
4. **Provides transparent operation** - no user intervention needed

**No troubleshooting required** - everything works out of the box!

## Directory Structure

- **PostgreSQL Data**: `/var/lib/postgresql/data`
- **pgBackRest Repository**: `/var/lib/pgbackrest` 
- **pgBackRest Logs**: `/var/log/pgbackrest`
- **pgBackRest Config**: `/etc/pgbackrest/pgbackrest.conf`

## Environment Variables

### Required S3 Configuration
- `S3_BUCKET_NAME`: S3 bucket for backups
- `S3_ENDPOINT`: S3 endpoint URL
- `S3_ACCESS_KEY`: S3 access key
- `S3_ACCESS_KEY_SECRET`: S3 secret key  
- `S3_REGION`: S3 region

### PostgreSQL Configuration
- `POSTGRES_PASSWORD`: Required - PostgreSQL superuser password
- `POSTGRES_USER`: Optional - PostgreSQL superuser name (default: postgres)
- `POSTGRES_DB`: Optional - Database name to create (default: postgres)
- `PGDATA`: Optional - PostgreSQL data directory (default: /var/lib/postgresql/data)

## Architecture

- PostgreSQL configured with WAL archiving to pgBackRest
- pgBackRest configured to store backups and archives in S3-compatible storage
- Automatic stanza creation and recreation during initialization
- Intelligent wrapper handles all stanza validation with automatic recovery
- Works with ephemeral containers - no persistent storage required for basic operation