.PHONY: run test race vet tidy build

run:
	go run ./cmd/collarsim

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy

build:
	go build -o bin/collarsim ./cmd/collarsim
