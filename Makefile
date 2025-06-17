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
build:	## build go binary to current directory.
	@go build cmd/mym/mym.go
	
.PHONY: install
install:	## install go binary to $GOPATH/bin.
	@go install cmd/mym/mym.go

##@ Usage
.PHONY: run
run:	## run data and save it into /data directory.
	@go run cmd/mym/mym.go run

.PHONY: docker-build
docker-build:	## build docker image.
	$(DOCKER) build -t $(NAME) . -f docker/Dockerfile

.PHONY: docker-run
docker-run:	## run local development server in docker.
	@$(DOCKER) stop $(NAME) || true && $(DOCKER) rm $(NAME) || true
	$(DOCKER) run -e GITHUB_TOKEN=$(GITHUB_TOKEN) --name $(NAME) $(NAME)


##@ Utility
.PHONY: sqlitetocsv
sqlitetocsv:	## convert data from sqlite3 to csv.
	@if [ -z $(SQLITE) ]; then echo "SQLite3 could not be found. See https://www.sqlite.org/download.html"; exit 2; fi
	sqlite3 -header -csv data/michelin.db "SELECT r.name as Name, r.address as Address, r.location as Location, ra.price as Price, r.cuisine as Cuisine, r.longitude as Longitude, r.latitude as Latitude, r.phone_number as PhoneNumber, r.url as Url, r.website_url as WebsiteUrl, ra.distinction as Award, ra.green_star as GreenStar, r.facilities_and_services as FacilitiesAndServices, r.description as Description FROM restaurants r JOIN restaurant_awards ra ON r.id = ra.restaurant_id WHERE ra.year = (SELECT MAX(year) FROM restaurant_awards ra2 WHERE ra2.restaurant_id = r.id);" > data/michelin_my_maps.csv
