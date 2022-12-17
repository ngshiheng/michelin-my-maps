NAME := michelin-my-maps
DOCKER := $(shell command -v docker 2> /dev/null)
MILLER := $(shell command -v mlr 2> /dev/null)

.DEFAULT_GOAL := help
.PHONY: help
help:  ## display this help message.
	@awk 'BEGIN {FS = ":.*##"; printf "\n\
	Usage:\n\
	  make \033[36m<target>\033[0m\n\
	\n\
	Targets:\n\
	"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n\
	", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: test
test:	## run all the tests.
	@go test ./... -v -count=1

.PHONY: lint
lint:	## run lint with golangci-lint in docker.
	@if [ -z $(DOCKER) ]; then echo "Docker could not be found. See https://docs.docker.com/"; exit 2; fi
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
	
.PHONY: build
build:	## build binary.
	@go build -o michelin-my-maps cmd/app/main.go

.PHONY: run
run:	## go run main.go.
	@go run cmd/app/main.go


.PHONY: csvtojson
csvtojson:	## convert dataset from csv to json.
	@if [ -z $(MILLER) ]; then echo "Miller could not be found. See https://github.com/johnkerl/miller"; exit 2; fi
	@mlr --c2j --jlistwrap cat data/michelin_my_maps.csv > docs/data.json
