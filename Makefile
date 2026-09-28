-include .env.example
-include .env
export

generate:
	go tool oapi-codegen \
	  -generate types,chi-server \
	  -package api \
	  -include-operation-ids createTrip,getTrip,finishTrip,health,ready \
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