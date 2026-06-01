.PHONY: run build cli-build test lint docker-up docker-down migrate coverage

run:
	go run cmd/api/main.go

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api cmd/api/main.go

cli-build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/wapgo cmd/cli/main.go

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate:
	go run cmd/api/main.go migrate

tidy:
	go mod tidy

sec:
	gosec ./...
	govulncheck ./...
