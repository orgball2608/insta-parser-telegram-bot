dev:
	go run ./cmd/main.go

tests:
	go test -parallel=20 -covermode atomic -coverprofile=coverage.out ./...

build:
	rm ./build-out || true
	go build -ldflags="-s -w" -o build-out cmd/main.go

docker-build:
	docker-compose up --build bot

docker-up:
	docker-compose up -d
