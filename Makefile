NAME := michelin-my-maps
DOCKER := $(shell command -v docker 2> /dev/null)
MILLER := $(shell command -v mlr 2> /dev/null)
DATASETTE := $(shell command -v datasette 2> /dev/null)
SQLITE := $(shell command -v sqlite3 2> /dev/null)

.DEFAULT_GOAL := help
.PHONY: help
help:   ## display this help message.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m\t%s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development
.PHONY: test
test:   ## run all the tests.
	@go test ./... -count=1 | grep -v 'no test files'

.PHONY: lint
lint:   ## run lint with golangci-lint in docker.
	@if [ -z $(DOCKER) ]; then echo "Docker could not be found. See https://docs.docker.com/"; exit 2; fi
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
	
.PHONY: build
build:  ## build go binary to bin/.
	@go build -o bin/ cmd/mym/mym.go
	
.PHONY: install
install:    ## install go binary to $GOPATH/bin.
	@go install cmd/mym/mym.go

##@ Usage
.PHONY: scrape
scrape: ## scrape data and save it into /data directory.
	@go run cmd/mym/mym.go scrape

.PHONY: datasette
datasette:  ## run datasette with metadata.json for local development.
	@if [ -z $(DATASETTE) ]; then echo "Datasette could not be found. See https://docs.datasette.io/en/stable/installation.html"; exit 2; fi
	@PORT=8001; \
	while lsof -iTCP:$$PORT -sTCP:LISTEN >/dev/null 2>&1; do \
	  PORT=$$((PORT+1)); \
	done; \
	echo "Starting datasette on port $$PORT"; \
	$(DATASETTE) --root data/michelin.db --metadata docker/datasette/metadata.json --port $$PORT

##@ Docker
.PHONY: docker-build-scraper
docker-build-scraper:   ## build scraper docker image.
	$(DOCKER) build -t mym-scraper . -f docker/scraper/Dockerfile

.PHONY: docker-run-scraper
docker-run-scraper: ## run scraper docker container.
	@$(DOCKER) stop mym-scraper || true && $(DOCKER) rm mym-scraper || true
	$(DOCKER) run \
        -e DATASETTE_SERVICE_ID=$(DATASETTE_SERVICE_ID) \
        -e GITHUB_TOKEN=$(GITHUB_TOKEN) \
        -e MINIO_ACCESS_KEY=$(MINIO_ACCESS_KEY) \
        -e MINIO_BUCKET=$(MINIO_BUCKET) \
        -e MINIO_ENDPOINT=$(MINIO_ENDPOINT) \
        -e MINIO_SECRET_KEY=$(MINIO_SECRET_KEY) \
        -e MYM_EMAIL=$(MYM_EMAIL) \
        -e MYM_PASSWORD=$(MYM_PASSWORD) \
        -e RAILWAY_API_TOKEN=$(RAILWAY_API_TOKEN) \
        --name mym-scraper mym-scraper

.PHONY: docker-build-datasette
docker-build-datasette:   ## build datasette docker image.
	@if [ -z $(DOCKER) ]; then echo "Docker could not be found. See https://docs.docker.com/"; exit 2; fi
	$(DOCKER) build -t mym-datasette . -f docker/datasette/Dockerfile --platform linux/amd64

##@ Utility
.PHONY: sqlitetocsv
sqlitetocsv:    ## convert data from sqlite3 to csv.
	@if [ -z $(SQLITE) ]; then echo "SQLite3 could not be found. See https://www.sqlite.org/download.html"; exit 2; fi
	sqlite3 -header -csv data/michelin.db "SELECT r.name as Name, r.address as Address, r.location as Location, ra.price as Price, r.cuisine as Cuisine, r.longitude as Longitude, r.latitude as Latitude, r.phone_number as PhoneNumber, r.url as Url, r.website_url as WebsiteUrl, ra.distinction as Award, ra.green_star as GreenStar, r.facilities_and_services as FacilitiesAndServices, r.description as Description FROM restaurants r JOIN restaurant_awards ra ON r.id = ra.restaurant_id WHERE ra.year = ( SELECT MAX(year) FROM restaurant_awards ra2 WHERE ra2.restaurant_id = r.id ) AND DATE(r.updated_at) = DATE('now');" > data/michelin_my_maps.csv
