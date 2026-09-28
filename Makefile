-include .env
export

generate:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

	go tool oapi-codegen \
	  -generate types,chi-server \
	  -package api \
	  -o internal/generated/api.gen.go \
	  contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir ./migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir ./migrations postgres "$(DATABASE_URL)" down

migrate-status:
	go tool goose -dir ./migrations postgres "$(DATABASE_URL)" status

run:
	go run ./cmd/trip-service

build:
	go build -o bin/trip-service ./cmd/trip-service