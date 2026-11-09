.PHONY: build run test test-integration generate migrate-up migrate-down tidy fmt lint
.PHONY: build run test generate migrate-up migrate-down tidy fmt lint

-include .env
export

DATABASE_URL ?= $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)

build:
	go build ./...

run:
	go run ./cmd/trip-service

test:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -tags=integration ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal

lint:
	go vet ./...

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate-up:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down
