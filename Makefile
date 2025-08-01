.PHONY: build run test docker-up docker-down

build:
	go build -o bin/server ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down