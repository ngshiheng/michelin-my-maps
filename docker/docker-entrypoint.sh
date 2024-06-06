#!/usr/bin/env/sh

set -e

main() {
    check
    run_mym
    sqlitetocsv
    publish_to_vercel
    publish_to_github
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

    if [ -z "$GITHUB_TOKEN" ]; then
        echo "Error: GITHUB_TOKEN is not set. Please set it before running this script."
        exit 1
    fi

    check_cli_installed curl
    check_cli_installed datasette
    check_cli_installed jq
    check_cli_installed mym
    check_cli_installed sqlite3
    check_cli_installed vercel

    curl https://api.incolumitas.com/ | jq
    echo "All checks passed."
}

run_mym() {
    echo "Running mym..."
    rm -rf cache/ michelin.db
    mym -log error
    if [ ! -f michelin.db ]; then
        echo "Error: michelin.db does not exist. Exiting..."
        exit 1
    fi
}

sqlitetocsv() {
    echo "Converting SQLite data to CSV..."
    if [ ! -f michelin.db ]; then
        echo "Error: michelin.db does not exist. Cannot convert to CSV. Exiting..."
        exit 1
    fi
    mkdir -p data
    sqlite3 -header -csv michelin.db "SELECT name as Name, address as Address, location as Location, price as Price, cuisine as Cuisine, longitude as Longitude, latitude as Latitude, phone_number as PhoneNumber, url as Url, website_url as WebsiteUrl, distinction as Award, green_star as GreenStar, facilities_and_services as FacilitiesAndServices, description as Description from restaurants;" >data/michelin_my_maps.csv
}

publish_to_github() {
    echo "Publishing new CSV to GitHub..."
    ENCODED_CSV_CONTENT=$(cat data/michelin_my_maps.csv | base64)

    CURRENT_SHA=$(curl -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        https://api.github.com/repos/ngshiheng/michelin-my-maps/contents/data/michelin_my_maps.csv | jq -r '.sha')

    echo '{"message":"chore(data): update generated csv", "content":"'"$ENCODED_CSV_CONTENT"'", "sha":"'"$CURRENT_SHA"'"}' |
        curl -X PUT -H "Authorization: token $GITHUB_TOKEN" \
            -d @- \
            https://api.github.com/repos/ngshiheng/michelin-my-maps/contents/data/michelin_my_maps.csv
}

publish_to_vercel() {
    echo "Publishing datasette to Vercel..."
    datasette publish vercel michelin.db --project=michelin-my-maps --install=datasette-cluster-map --install=datasette-hashed-urls --token="$VERCEL_TOKEN" --metadata metadata.json --setting allow_download off --extra-options "-i"
}

main "$@"
