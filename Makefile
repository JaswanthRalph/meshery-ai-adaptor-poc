.PHONY: build test lint fmt run

build:
	go build -o meshery-ai-adapter .
	go build -o mesheryctl-ai ./cmd/mesheryctl-ai/

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

run:
	go run .
