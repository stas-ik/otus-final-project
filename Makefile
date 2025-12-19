.PHONY: build test lint docker-build run-test-race

BINARY_NAME=gomigrator
GO_VERSION=1.25

build:
	go build -o $(BINARY_NAME) ./cmd/gomigrator/main.go

test:
	go test ./...

run-test-race:
	go test -race -count 100 ./...

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(BINARY_NAME) .

all: lint run-test-race build
