ifneq (,$(wildcard ./.env))
    include .env
    export
endif

dev:
	go run ./cmd/main.go

lint:
	golangci-lint run

tests:
	go test -parallel=20 -covermode atomic -coverprofile=coverage.out ./...

build:
	rm ./build-out || true
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build-out cmd/main.go
	upx -9 -q ./build-out

docker-build:
	docker-compose up --build

docker-up:
	docker-compose up -d
