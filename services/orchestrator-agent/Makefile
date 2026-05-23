.PHONY: build run test lint docker-build

SERVICE := orchestrator-agent
IMAGE := antigravity/$(SERVICE):latest

build:
	go build -o bin/$(SERVICE) ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p 8084:8084 -p 50052:50052 $(IMAGE)
