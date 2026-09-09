.PHONY: run-backend run-sim test race vet tidy build up down migrate

run-backend:
	go run ./cmd/backend

run-sim:
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
	go build -o bin/backend ./cmd/backend
	go build -o bin/collarsim ./cmd/collarsim

up:
	docker compose up -d

down:
	docker compose down

migrate:
	GOOSE_DRIVER=postgres \
	GOOSE_DBSTRING="postgres://paddock:paddock@localhost:5432/paddock?sslmode=disable" \
	goose -dir db/migrations up
