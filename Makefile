NAME := michelin-my-maps
DOCKER := $(shell command -v docker 2> /dev/null)

.DEFAULT_GOAL := help
.PHONY: help
help:
	@echo "Use 'make <target>' where <target> is one of:"
	@echo ""
	@echo "  help		show this help message"
	@echo "  test		run all the tests"
	@echo "  lint		run lint with golangci-lint in docker"
	@echo "  build		build $(NAME) binary"
	@echo "  run		go run $(NAME)"
	@echo ""
	@echo "Check the Makefile to know exactly what each target is doing."

.PHONY: test
test:
	go test ./... -v -count=1

.PHONY: lint
lint:
	@if [ -z $(DOCKER) ]; then echo "Docker could not be found. See https://docs.docker.com/"; exit 2; fi
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
	
build:
	go build -o bin/main .

run:
	go run main.go
