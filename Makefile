test:
	go test ./... -v -count=1

lint:
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
	
build:
	go build -o bin/main .

run:
	go run main.go
