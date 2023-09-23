.PHONE: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'




.PHONY: static-check
static-check: ## Run static code checks
	golangci-lint run

.PHONY: generate
generate: ## Generate all code to run the service
	go generate ./...
	@# the experimental flag is required for pgx compatible code, see: https://docs.sqlc.dev/en/stable/guides/using-go-and-pgx.html?highlight=experimental#getting-started
	sqlc generate --experimental

.PHONY: test
test: static-check generate test-unit test-integration ## Run all tests
	go tool cover -html=cover.out -o cover.html; xdg-open cover.html
	go tool cover -func cover.out | grep total:


.PHONY: test-unit
test-unit:
	go test -race ./... -coverprofile cover.out

.PHONY: test-integration
test-integration:
	go test -race --tags="integration" ./... -coverprofile cover.out




.PHONY: upgrade
upgrade:
	go get -t -u ./...
	go mod tidy

.PHONY: install-tools
install-tools: ## Initialise this machine with development dependencies
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b $(go env GOPATH)/bin v1.51.1
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/kyleconroy/sqlc/cmd/sqlc@latest




.PHONY: run
run: ## Run all dependencies inside docker containers
	docker-compose pull
	docker-compose up -d
	xdg-open http://localhost:8081 # open pgadmin in the browser
	#xdg-open http://localhost:9090 # open prometheus in the browser
	xdg-open http://localhost:3000 # open grafana in the browser

.PHONY: db
db: ## Connect to the database inside the running docker container
	PGPASSWORD=secret psql -U arrower -d arrower -h localhost