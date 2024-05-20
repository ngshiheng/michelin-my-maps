NAME := michelin-my-maps
DOCKER := $(shell command -v docker 2> /dev/null)
MILLER := $(shell command -v mlr 2> /dev/null)
PYTHON := $(shell command -v python3 2> /dev/null)
SQLITE := $(shell command -v sqlite3 2> /dev/null)

.DEFAULT_GOAL := help
.PHONY: help
help:  ## display this help message.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development
.PHONY: test
test:	## run all the tests.
	@go test ./... -v -count=1

.PHONY: lint
lint:	## run lint with golangci-lint in docker.
	@if [ -z $(DOCKER) ]; then echo "Docker could not be found. See https://docs.docker.com/"; exit 2; fi
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
	
.PHONY: build
build:	## build go binary.
	@go build cmd/mym/mym.go
	

##@ Usage
.PHONY: crawl
crawl:	## crawl data and save it into /data directory.
	@rm -rf michelin.db
	@go run cmd/mym/mym.go

.PHONY: docker-build
docker-build:	## build docker image.
	$(DOCKER) build -t $(NAME) . -f docker/Dockerfile

.PHONY: docker-run
docker-run:	## run local development server in docker.
	@$(DOCKER) stop $(NAME) || true && $(DOCKER) rm $(NAME) || true
	$(DOCKER) run -e VERCEL_TOKEN=$(VERCEL_TOKEN) -e GITHUB_TOKEN=$(GITHUB_TOKEN) --name $(NAME) $(NAME)


##@ Utility
.PHONY: sqlitetocsv
sqlitetocsv:	## convert data from sqlite3 to csv.
	@if [ -z $(SQLITE) ]; then echo "SQLite3 could not be found. See https://www.sqlite.org/download.html"; exit 2; fi
	sqlite3 -header -csv michelin.db "SELECT name as Name, address as Address, location as Location, price as Price, cuisine as Cuisine, longitude as Longitude, latitude as Latitude, phone_number as PhoneNumber, url as Url, website_url as WebsiteUrl, distinction as Award, green_star as GreenStar, facilities_and_services as FacilitiesAndServices, description as Description from restaurants;" > data/michelin_my_maps.csv

.PHONY: sqlitetojson
sqlitetojson:	## convert data from sqlite3 to json.
	@if [ -z $(SQLITE) ]; then echo "SQLite3 could not be found. See https://www.sqlite.org/download.html"; exit 2; fi
	sqlite3 michelin.db '.mode json' '.once docs/data.json' 'SELECT name as Name, address as Address, location as Location, price as Price, cuisine as Cuisine, longitude as Longitude, latitude as Latitude, phone_number as PhoneNumber, url as Url, website_url as WebsiteUrl, distinction as Award, green_star as GreenStar,facilities_and_services as FacilitiesAndServices, description as Description from restaurants;'

.PHONY: csvtojson
csvtojson:	## convert data from csv to json.
	@if [ -z $(MILLER) ]; then echo "Miller could not be found. See https://github.com/johnkerl/miller"; exit 2; fi
	mlr --c2j --jlistwrap then put 'for (k, v in $$*) { $$[k] = string(v) }' then cat data/michelin_my_maps.csv > docs/data.json
