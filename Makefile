.PHONY: build lint test test-cover tidy

build:
	go build ./...

lint:
	golangci-lint run

test:
	go test ./...

test-cover:
	go test -cover ./...

tidy:
	go mod tidy
