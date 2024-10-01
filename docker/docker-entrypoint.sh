#!/bin/sh

set -eu

# Configuration
CSV_FILE="data/michelin_my_maps.csv"
DB_FILE="michelin.db"
MIN_CSV_LINES=15000
GITHUB_REPO="ngshiheng/michelin-my-maps"
VERCEL_PROJECT="michelin-my-maps"

REQUIRED_TOOLS="curl datasette jq mym sqlite3 vercel"

# Main function
main() {
    check_environment
    check_dependencies
    run_mym
    convert_sqlite_to_csv
    publish_to_github
    # publish_to_vercel  # Uncomment if needed
}

# Environment and dependency checks
check_environment() {
    check_env_var "VERCEL_TOKEN"
    check_env_var "GITHUB_TOKEN"
}

check_dependencies() {
    for tool in $REQUIRED_TOOLS; do
        check_cli_installed "$tool"
    done
    curl https://api.incolumitas.com/ | jq
    echo "All checks passed."
}

check_env_var() {
    if [ -z "$(eval echo \$$1)" ]; then
        echo "Error: $1 is not set. Please set it before running this script."
        exit 1
    fi
}

check_cli_installed() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo >&2 "Error: $1 is not installed."
        exit 1
    fi
}

# Scrape and conversion functions
run_mym() {
    echo "Running mym..."
    rm -rf cache/ "$DB_FILE"
    mym -log error
    if [ ! -f "$DB_FILE" ]; then
        echo "Error: $DB_FILE does not exist. Exiting..."
        exit 1
    fi
}

convert_sqlite_to_csv() {
    echo "Converting SQLite data to CSV..."
    if [ ! -f "$DB_FILE" ]; then
        echo "Error: $DB_FILE does not exist. Cannot convert to CSV. Exiting..."
        exit 1
    fi
    mkdir -p "$(dirname "$CSV_FILE")"
    sqlite3 -header -csv "$DB_FILE" "SELECT name as Name, address as Address, location as Location, price as Price, cuisine as Cuisine, longitude as Longitude, latitude as Latitude, phone_number as PhoneNumber, url as Url, website_url as WebsiteUrl, distinction as Award, green_star as GreenStar, facilities_and_services as FacilitiesAndServices, description as Description from restaurants;" >"$CSV_FILE"
}

# Publishing functions
publish_to_github() {
    echo "Checking CSV before publishing to GitHub..."
    if ! check_csv_lines; then
        echo "CSV check failed. Skipping GitHub publish."
        return 1
    fi

    echo "Publishing new CSV to GitHub..."
    encoded_content=$(base64 <"$CSV_FILE")

    current_sha=$(curl -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$GITHUB_REPO/contents/$CSV_FILE" | jq -r '.sha')

    json_payload=$(printf '{"message": "chore(data): update generated csv", "content": "%s", "sha": "%s"}' "$encoded_content" "$current_sha")

    curl -X PUT -H "Authorization: token $GITHUB_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$json_payload" \
        "https://api.github.com/repos/$GITHUB_REPO/contents/$CSV_FILE"
}

publish_to_vercel() {
    echo "Publishing datasette to Vercel..."
    datasette publish vercel "$DB_FILE" \
        --project="$VERCEL_PROJECT" \
        --install=datasette-cluster-map \
        --install=datasette-hashed-urls \
        --token="$VERCEL_TOKEN" \
        --metadata metadata.json \
        --setting allow_download off \
        --setting allow_csv_stream off \
        --extra-options "-i"
}

# Helper functions
check_csv_lines() {
    echo "Checking CSV file line count..."

    if [ ! -f "$CSV_FILE" ]; then
        echo "Error: $CSV_FILE does not exist. Cannot check line count."
        return 1
    fi

    line_count=$(wc -l <"$CSV_FILE")

    if [ "$line_count" -lt "$MIN_CSV_LINES" ]; then
        echo "Error: $CSV_FILE has only $line_count lines. Minimum required is $MIN_CSV_LINES."
        return 1
    else
        echo "CSV file check passed. Line count: $line_count"
        return 0
    fi
}

# Run the main function
main "$@"
