# Common tasks. Everything runs in Docker; nothing needs installing on the host.

# Go toolchain in a container on the compose network, so tests can reach Postgres.
GO = docker run --rm --network ford-retain -u $$(id -u):$$(id -g) \
	-e HOME=/tmp -e GOCACHE=/go/cache -e GOMODCACHE=/go/pkg/mod \
	-e TEST_DATABASE_URL="postgres://ford:ford@postgres:5432/ford_retain?sslmode=disable" \
	-v ford-retain-go-cache:/go -v "$(CURDIR)":/app -w /app golang:1.27-alpine

.PHONY: dev down test fmt build prod

dev: ## API on http://localhost:8080 with Postgres and the mock Ford API
	docker compose up

down:
	docker compose down

test: ## Everything CI runs: gofmt, go vet, go test (Postgres tests included)
	docker compose up -d --wait postgres
	$(GO) sh -c 'test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }; go vet ./... && go test ./...'

fmt:
	$(GO) gofmt -w .

build: ## Build the production image
	docker compose -f compose.prod.yaml build

prod: ## Run the production image with Postgres
	docker compose -f compose.prod.yaml up --build
