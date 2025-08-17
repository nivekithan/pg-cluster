#!/bin/bash
set -e

# Script to update PostgreSQL secrets imperatively using kubectl
# This script validates all required environment variables and updates an existing Kubernetes secret

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Updates an existing Kubernetes secret 'postgres-secrets' with all required environment variables"
    echo "for the postgres-pgbackrest container."
    echo ""
    echo "Required environment variables:"
    echo "  PostgreSQL Configuration:"
    echo "    POSTGRES_PASSWORD    - PostgreSQL superuser password"
    echo "    POSTGRES_USER        - PostgreSQL superuser username"
    echo "    POSTGRES_DB          - Default database name"
    echo "    PGDATA              - PostgreSQL data directory path"
    echo ""
    echo "  S3 Configuration:"
    echo "    S3_BUCKET_NAME      - S3 bucket for pgBackRest backups"
    echo "    S3_ENDPOINT         - S3 endpoint URL"
    echo "    S3_ACCESS_KEY       - S3 access key ID"
    echo "    S3_ACCESS_KEY_SECRET - S3 secret access key"
    echo "    S3_REGION           - S3 region"
    echo ""
    echo "  pgBackRest Configuration:"
    echo "    REPO1_RETENTION_FULL - Full backup retention count"
    echo "    STANZA_NAME         - pgBackRest stanza name"
    echo "    ARCHIVE_TIMEOUT     - WAL archive timeout"
    echo "    MAX_WAL_SENDERS     - Maximum WAL sender processes"
    echo "    WAL_KEEP_SIZE       - WAL retention size"
    echo "    PG1_SOCKET_PATH     - PostgreSQL Unix socket directory"
    echo ""
    echo "Options:"
    echo "  -n, --namespace NAMESPACE   Kubernetes namespace (default: default)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "Example:"
    echo "  export POSTGRES_PASSWORD='mypassword'"
    echo "  export POSTGRES_USER='postgres'"
    echo "  # ... set other variables ..."
    echo "  $0 --namespace postgres-system"
}

validate_required_env() {
    local missing_vars=()

    # All required environment variables (matching validate-env.sh)
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
        echo "❌ Error: Required environment variables are missing:"
        printf "  - %s\n" "${missing_vars[@]}"
        echo ""
        echo "Please set all required environment variables before running this script."
        echo "Use '$0 --help' for more information."
        return 1
    fi

    echo "✅ Environment validation successful - all required variables present"
    return 0
}

check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        echo "❌ Error: kubectl is not installed or not in PATH"
        echo "Please install kubectl before running this script."
        return 1
    fi

    # Check if we can connect to the cluster
    if ! kubectl cluster-info &> /dev/null; then
        echo "❌ Error: Cannot connect to Kubernetes cluster"
        echo "Please check your kubeconfig and cluster connectivity."
        return 1
    fi

    echo "✅ kubectl is available and connected to cluster"
    return 0
}

validate_secret_exists() {
    local namespace="$1"
    local secret_name="postgres-secrets"

    echo "🔍 Checking if secret '$secret_name' exists in namespace '$namespace'..."

    if ! kubectl get secret "$secret_name" -n "$namespace" &> /dev/null; then
        echo "❌ Error: Secret '$secret_name' does not exist in namespace '$namespace'"
        echo "Please create the secret first using:"
        echo "  kubectl apply -f manifests/postgres-secrets.yaml"
        echo "Or create it with:"
        echo "  kubectl create secret generic $secret_name --namespace=$namespace --from-literal=placeholder=value"
        return 1
    fi

    echo "✅ Secret '$secret_name' found in namespace '$namespace'"
    return 0
}

update_secret() {
    local namespace="$1"
    local secret_name="postgres-secrets"

    echo "🔧 Updating Kubernetes secret '$secret_name' in namespace '$namespace'..."

    # Create a temporary patch file with all the data
    local patch_data=$(cat <<EOF
{
  "data": {
    "POSTGRES_PASSWORD": "$(echo -n "$POSTGRES_PASSWORD" | base64 -w 0)",
    "POSTGRES_USER": "$(echo -n "$POSTGRES_USER" | base64 -w 0)",
    "POSTGRES_DB": "$(echo -n "$POSTGRES_DB" | base64 -w 0)",
    "PGDATA": "$(echo -n "$PGDATA" | base64 -w 0)",
    "S3_BUCKET_NAME": "$(echo -n "$S3_BUCKET_NAME" | base64 -w 0)",
    "S3_ENDPOINT": "$(echo -n "$S3_ENDPOINT" | base64 -w 0)",
    "S3_ACCESS_KEY": "$(echo -n "$S3_ACCESS_KEY" | base64 -w 0)",
    "S3_ACCESS_KEY_SECRET": "$(echo -n "$S3_ACCESS_KEY_SECRET" | base64 -w 0)",
    "S3_REGION": "$(echo -n "$S3_REGION" | base64 -w 0)",
    "REPO1_RETENTION_FULL": "$(echo -n "$REPO1_RETENTION_FULL" | base64 -w 0)",
    "STANZA_NAME": "$(echo -n "$STANZA_NAME" | base64 -w 0)",
    "ARCHIVE_TIMEOUT": "$(echo -n "$ARCHIVE_TIMEOUT" | base64 -w 0)",
    "MAX_WAL_SENDERS": "$(echo -n "$MAX_WAL_SENDERS" | base64 -w 0)",
    "WAL_KEEP_SIZE": "$(echo -n "$WAL_KEEP_SIZE" | base64 -w 0)",
    "PG1_SOCKET_PATH": "$(echo -n "$PG1_SOCKET_PATH" | base64 -w 0)"
  }
}
EOF
)

    # Apply the patch to update the secret
    if echo "$patch_data" | kubectl patch secret "$secret_name" -n "$namespace" --type='merge' --patch-file=/dev/stdin; then
        echo "✅ Secret '$secret_name' updated successfully in namespace '$namespace'"
        echo ""
        echo "Updated values:"
        echo "  - PostgreSQL Database: $POSTGRES_DB"
        echo "  - PostgreSQL User: $POSTGRES_USER"
        echo "  - PostgreSQL Socket Path: $PG1_SOCKET_PATH"
        echo "  - pgBackRest Stanza: $STANZA_NAME"
        echo "  - S3 Bucket: $S3_BUCKET_NAME"
        echo "  - S3 Region: $S3_REGION"
        echo ""
        echo "The secret is ready for use in your PostgreSQL deployment."
    else
        echo "❌ Failed to update secret"
        return 1
    fi
}

# Main script execution
main() {
    local namespace="default"

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                namespace="$2"
                shift 2
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                echo "❌ Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    echo "🚀 PostgreSQL Secrets Updater"
    echo "============================="
    echo "Target namespace: $namespace"
    echo ""

    # Validate prerequisites
    if ! validate_required_env; then
        exit 1
    fi

    if ! check_kubectl; then
        exit 1
    fi

    # Validate secret exists
    if ! validate_secret_exists "$namespace"; then
        exit 1
    fi

    # Update the secret
    if ! update_secret "$namespace"; then
        exit 1
    fi

    echo ""
    echo "🎉 PostgreSQL secrets have been successfully updated!"
}

# Run main function with all arguments
main "$@"