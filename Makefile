.PHONY: build lint fmt test test-coverage clean

BINARY := mdp
MAIN := ./cmd/mdp

## Build the binary.
build:
	go build -o $(BINARY) $(MAIN)

## Run golangci-lint.
lint:
	golangci-lint run

## Format code via golangci-lint formatters.
fmt:
	golangci-lint fmt

## Run all tests.
test:
	go test ./...

## Run tests with coverage output.
test-coverage:
	go test -race -coverprofile=coverage.out ./...

## Remove build artifacts.
clean:
	rm -f $(BINARY) coverage.out
