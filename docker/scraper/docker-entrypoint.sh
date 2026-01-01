#!/bin/sh

set -eu

# Configuration
CSV_FILE="data/michelin_my_maps.csv"
DB_FILE="data/michelin.db"
MIN_CSV_LINES=17000

REQUIRED_TOOLS="curl jq mym sqlite3 mc"

# Main function
main() {
    check_environment
    check_dependencies
    download_from_minio
    run_mym
    upload_to_minio
    trigger_datasette_redeploy
    convert_sqlite_to_csv
    publish_to_github
}

# Environment and dependency checks
check_environment() {
    check_env_var "GITHUB_TOKEN"
}

check_dependencies() {
    for tool in $REQUIRED_TOOLS; do
        check_cli_installed "$tool"
    done
    # curl https://api.incolumitas.com/ | jq # FIXME: endpoint not available
    echo "check environment and dependencies passed"
}

check_env_var() {
    if [ -z "$(eval echo \$$1)" ]; then
        echo "error: $1 is not set. set it before running this script."
        exit 1
    fi
}

check_cli_installed() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo >&2 "error: $1 is not installed."
        exit 1
    fi
}

download_from_minio() {
    echo "download $DB_FILE from MinIO (if exists)"
    mkdir -p ./data/
    if [ -z "${MINIO_ENDPOINT:-}" ] || [ -z "${MINIO_ACCESS_KEY:-}" ] || [ -z "${MINIO_SECRET_KEY:-}" ] || [ -z "${MINIO_BUCKET:-}" ]; then
        echo "error: MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, and MINIO_BUCKET must be set."
        exit 1
    fi
    mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
    if mc ls "minio/$MINIO_BUCKET/michelin.db" >/dev/null 2>&1; then
        mc cp "minio/$MINIO_BUCKET/michelin.db" "./data/michelin.db"
        echo "downloaded existing DB file from MinIO to ./data"
    else
        echo "no existing DB file found in MinIO, start fresh"
    fi
}

# Trigger Datasette redeploy on Railway
trigger_datasette_redeploy() {
    if [ -z "${RAILWAY_API_TOKEN:-}" ] || [ -z "${DATASETTE_SERVICE_ID:-}" ]; then
        echo "skip Datasette redeploy: RAILWAY_API_TOKEN or DATASETTE_SERVICE_ID not set"
        return 0
    fi
    echo "trigger Datasette redeploy on Railway"
    curl -X POST "https://backboard.railway.app/project/${DATASETTE_SERVICE_ID}/deployments" \
        -H "Authorization: Bearer $RAILWAY_API_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{}'
}

# Upload SQLite to MinIO
upload_to_minio() {
    echo "upload $DB_FILE to MinIO"
    if [ -z "${MINIO_ENDPOINT:-}" ] || [ -z "${MINIO_ACCESS_KEY:-}" ] || [ -z "${MINIO_SECRET_KEY:-}" ] || [ -z "${MINIO_BUCKET:-}" ]; then
        echo "error: MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, and MINIO_BUCKET must be set."
        exit 1
    fi
    mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
    mc cp "$DB_FILE" "minio/$MINIO_BUCKET/$(basename "$DB_FILE")"
}

# Scraper function
run_mym() {
    echo "run mym"
    echo "database will be created at $DB_FILE"

    # Remove cache
    rm -rf cache/
    mym scrape -log warn
    if [ ! -f "$DB_FILE" ]; then
        echo "error: $DB_FILE does not exist. exit"
        exit 1
    fi
}

convert_sqlite_to_csv() {
    echo "convert SQLite data to CSV"
    if [ ! -f "$DB_FILE" ]; then
        echo "error: $DB_FILE does not exist. cannot convert to CSV. exit"
        exit 1
    fi
    mkdir -p "$(dirname "$CSV_FILE")"
    sqlite3 -header -csv "$DB_FILE" "SELECT r.name as Name, r.address as Address, r.location as Location, ra.price as Price, r.cuisine as Cuisine, r.longitude as Longitude, r.latitude as Latitude, r.phone_number as PhoneNumber, r.url as Url, r.website_url as WebsiteUrl, ra.distinction as Award, ra.green_star as GreenStar, r.facilities_and_services as FacilitiesAndServices, r.description as Description FROM restaurants r JOIN restaurant_awards ra ON r.id = ra.restaurant_id WHERE ra.year = ( SELECT MAX(year) FROM restaurant_awards ra2 WHERE ra2.restaurant_id = r.id ) AND DATE(r.updated_at) = DATE('now');" >"$CSV_FILE"
}

# Publishing functions
publish_to_github() {
    echo "check CSV before publish to GitHub"
    if ! check_csv_lines; then
        echo "CSV check failed. skip GitHub publish"
        return 1
    fi

    echo "publish new CSV to GitHub"
    encoded_content=$(base64 <"$CSV_FILE")

    current_sha=$(curl -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        https://api.github.com/repos/ngshiheng/michelin-my-maps/contents/data/michelin_my_maps.csv | jq -r '.sha')

    echo '{"message":"chore(data): update generated csv", "content":"'"$encoded_content"'", "sha":"'"$current_sha"'"}' |
        curl -X PUT -H "Authorization: token $GITHUB_TOKEN" \
            -d @- \
            https://api.github.com/repos/ngshiheng/michelin-my-maps/contents/data/michelin_my_maps.csv
}

# Helper functions
check_csv_lines() {
    echo "check CSV file line count"

    if [ ! -f "$CSV_FILE" ]; then
        echo "error: $CSV_FILE does not exist. cannot check line count"
        return 1
    fi

    line_count=$(wc -l <"$CSV_FILE")

    if [ "$line_count" -lt "$MIN_CSV_LINES" ]; then
        echo "error: $CSV_FILE has only $line_count lines. minimum required is $MIN_CSV_LINES"
        return 1
    else
        echo "CSV file check passed. line count $line_count"
        return 0
    fi
}

# Run the main function
main "$@"
