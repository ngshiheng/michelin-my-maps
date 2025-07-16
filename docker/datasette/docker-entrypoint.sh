#!/bin/sh

set -eu

DB_FILE="/app/michelin.db"

# Check required env vars
if [ -z "${MINIO_ENDPOINT:-}" ] || [ -z "${MINIO_ACCESS_KEY:-}" ] || [ -z "${MINIO_SECRET_KEY:-}" ] || [ -z "${MINIO_BUCKET:-}" ]; then
    echo "Error: MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, and MINIO_BUCKET must be set."
    exit 1
fi

# Configure mc and download DB
mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
mkdir -p /app
if mc ls "minio/$MINIO_BUCKET/michelin.db" >/dev/null 2>&1; then
    mc cp "minio/$MINIO_BUCKET/michelin.db" "$DB_FILE"
    echo "Downloaded michelin.db from Minio."
else
    echo "No michelin.db found in Minio bucket. Exiting."
    exit 1
fi

# Start Datasette
exec datasette -i "$DB_FILE" --metadata /docker/datasette/metadata.json --host 0.0.0.0 --port 8001 --setting allow_download off --setting allow_csv_stream off --setting max_csv_mb 0
