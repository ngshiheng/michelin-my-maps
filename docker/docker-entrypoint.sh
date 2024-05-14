#!/usr/bin/env/sh

set -e

main() {
    check
    run_mym
    publish_to_vercel
}

check_cli_installed() {
    command -v "$1" >/dev/null 2>&1 || {
        echo >&2 "Error: $1 is not installed."
        return 1
    }
}

check() {
    if [ -z "$VERCEL_TOKEN" ]; then
        echo "Error: VERCEL_TOKEN is not set. Please set it before running this script."
        exit 1
    fi

    check_cli_installed mym
    check_cli_installed vercel
    check_cli_installed datasette
    echo "All checks passed."
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
    datasette publish vercel michelin.db --project=michelin-my-maps --install=datasette-cluster-map --install=datasette-hashed-urls --token="$VERCEL_TOKEN" --metadata metadata.json --setting allow_download off
}

main "$@"
