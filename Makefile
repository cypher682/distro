SHELL := /usr/bin/env bash

.PHONY: demo test vet build fmt down logs seed

demo:
	docker compose up --build

test:
	docker compose run --rm dev go test ./...

vet:
	docker compose run --rm dev go vet ./...

build:
	docker compose run --rm dev go build ./...

fmt:
	docker compose run --rm dev gofmt -w .

seed:
	docker compose exec db psql -U distro -d distro -f /docker-entrypoint-initdb.d/0005_seed.sql

down:
	docker compose down -v

logs:
	docker compose logs -f
