build:
	go build -a -installsuffix cgo -o app ./cmd

run:
	go run ./cmd/main.go

swag:
	swag init --parseDependency --parseInternal -g cmd/main.go 

# For test
test:
	ginkgo tests/

lint:
	golangci-lint run

# For docker
docker-build:
	docker compose up --build -d
