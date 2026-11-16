-include .env.example
-include .env
export

.PHONY: generate migrate migrate-down migrate-status run build lint test test-integration

generate:
	go generate ./internal/...
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose \
		-dir ./migrations \
		postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose \
		-dir ./migrations \
		postgres "$(DATABASE_URL)" down

migrate-status:
	go tool goose \
		-dir ./migrations \
		postgres "$(DATABASE_URL)" status

run:
	go run ./cmd/trip-service

build:
	mkdir -p bin
	go build -o bin/trip-service ./cmd/trip-service

lint:
	golangci-lint run ./...

test:
	go test \
		-race \
		-count=1 \
		-coverpkg=./internal/domain/...,./internal/usecase/... \
		-coverprofile=coverage.out \
		./...
	go tool cover -func=coverage.out

test-integration:
	go test \
		-race \
		-count=1 \
		-tags=integration \
		./internal/repository/...