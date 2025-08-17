#!/bin/bash
# Centralized environment variable validation - ALL REQUIRED

validate_required_env() {
    local missing_vars=()
    
    # All required environment variables
    local required_vars=(
        # PostgreSQL Configuration
        "POSTGRES_PASSWORD"
        "POSTGRES_USER"
        "POSTGRES_DB"
        "PGDATA"
        
        # S3 Configuration
        "S3_BUCKET_NAME"
        "S3_ENDPOINT"
        "S3_ACCESS_KEY"
        "S3_ACCESS_KEY_SECRET"
        "S3_REGION"
        
        # pgBackRest Configuration
        "REPO1_RETENTION_FULL"
        "STANZA_NAME"
        "ARCHIVE_TIMEOUT"
        "MAX_WAL_SENDERS"
        "WAL_KEEP_SIZE"
        "PG1_SOCKET_PATH"
    )
    
    # Check all required variables
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    # Report missing variables
    if [ ${#missing_vars[@]} -ne 0 ]; then
        echo "Error: Required environment variables are missing:"
        printf "  - %s\n" "${missing_vars[@]}"
        echo ""
        echo "All environment variables are required. See env.sample for complete list."
        return 1
    fi
    
    echo "Environment validation successful - all required variables present:"
    echo "  - PostgreSQL Database: $POSTGRES_DB"
    echo "  - PostgreSQL User: $POSTGRES_USER"
    echo "  - pgBackRest Stanza: $STANZA_NAME"
    echo "  - S3 Bucket: $S3_BUCKET_NAME"
    echo "  - S3 Region: $S3_REGION"
    return 0
}