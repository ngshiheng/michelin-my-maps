#!/bin/bash

set -e

main() {
    check
    run_mym
    publish_to_vercel
}

check() {
    echo "Checking..."
    if [ -z "$VERCEL_TOKEN" ]; then
        echo "Error: VERCEL_TOKEN is not set. Please set it before running this script."
        exit 1
    fi

    if ! command -v vercel &>/dev/null; then
        echo "Error: vercel is not installed. Please install it before running this script."
        exit 1
    fi

    if ! command -v datasette &>/dev/null; then
        echo "Error: datasette is not installed. Please install it before running this script."
        exit 1
    fi
}

run_mym() {
    echo "Running mym..."
    mym -log error
    if [ ! -f michelin.db ]; then
        echo "Error: michelin.db does not exist. Exiting..."
        exit 1
    fi
}

publish_to_vercel() {
    echo "Publishing datasette to Vercel..."
    datasette publish vercel michelin.db --project=michelin-my-maps --install=datasette-cluster-map --token="$VERCEL_TOKEN"
}

main "$@"
